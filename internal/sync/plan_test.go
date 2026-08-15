package sync

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/vzauartcc/roster-sync/internal/vatusa"
	"github.com/vzauartcc/roster-sync/internal/zau"
)

var fixedNow = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

func vController(cid int, membership string, roles []string, facility string) vatusa.Controller {
	controller := vatusa.Controller{
		CID:              cid,
		FName:            "F",
		LName:            "L",
		Email:            "f@example.com",
		Facility:         facility,
		Rating:           3,
		Membership:       membership,
		BroadcastOptedIn: true,
		FacilityJoinDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	for i, role := range roles {
		controller.Roles = append(controller.Roles, vatusa.Role{
			ID:        i + 1,
			CID:       cid,
			Facility:  "ZAU",
			Role:      role,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	}

	return controller
}

func zUser(cid int, member, visitor bool, roleCodes []string) zau.User {
	return zau.User{
		CID:                  cid,
		FName:                "F",
		LName:                "L",
		Email:                "f@example.com",
		Rating:               3,
		HomeFacility:         "ZAU",
		FlagBroadcastOptedIn: true,
		IsMember:             member,
		IsVisitor:            visitor,
		UseNamePrivacy:       false,
		RoleCodes:            roleCodes,
	}
}

func assertOp(t *testing.T, ops []Operation, kind OperationKind) {
	t.Helper()

	for _, operation := range ops {
		if operation.Kind == kind {
			return
		}
	}

	t.Errorf("expected op %s in ops %+v", kind, ops)
}

func assertNoOp(t *testing.T, ops []Operation, kind OperationKind) {
	t.Helper()

	for _, operation := range ops {
		if operation.Kind == kind {
			t.Errorf("expected no %s op, got %+v", kind, ops)
		}
	}
}

func TestPlanCreatesNewUserWithFilteredRoles(t *testing.T) {
	vUser := vController(1000001, "home", []string{"ATM", "C3"}, "ZAU")
	vUser.Roles = append(vUser.Roles,
		vatusa.Role{Facility: "BOS", Role: "ATM"},
		vatusa.Role{Facility: "ZHQ", Role: "ZZZ"},
	)

	ops := Plan(zau.Roster{}, []vatusa.Controller{vUser}, []string{"atm", "c3"}, fixedNow)

	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d: %+v", len(ops), ops)
	}

	operation := ops[0]
	if operation.Kind != OpCreate {
		t.Fatalf("expected OpCreate, got %s", operation.Kind)
	}

	want := []string{"atm", "c3"}
	if !slices.Equal(operation.Roles, want) {
		t.Errorf("expected roles %v, got %v", want, operation.Roles)
	}
}

func TestPlanCreatesVisitorWithRoles(t *testing.T) {
	vUser := vController(1000002, "visit", []string{"EC"}, "ZAU")

	ops := Plan(zau.Roster{}, []vatusa.Controller{vUser}, []string{"atm", "ec"}, fixedNow)

	if len(ops) != 1 || ops[0].Kind != OpCreate {
		t.Fatalf("expected 1 create op, got %+v", ops)
	}

	if !slices.Equal(ops[0].Roles, []string{"ec"}) {
		t.Errorf("expected visitor to receive roles, got %v", ops[0].Roles)
	}
}

func TestPlanNoOpsWhenNothingChanged(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, true, false, []string{"atm"})}}
	vUser := vController(1000001, "home", []string{"ATM"}, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm", "ec"}, fixedNow)

	if len(ops) != 0 {
		t.Errorf("expected no ops, got %+v", ops)
	}
}

func TestPlanEmitsCoreUpdateOnChange(t *testing.T) {
	z := zUser(1000001, true, false, []string{"atm"})
	z.FName = "Old"
	roster := zau.Roster{Home: []zau.User{z}}
	vUser := vController(1000001, "home", []string{"ATM"}, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm"}, fixedNow)

	assertOp(t, ops, OpUpdateCore)
}

func TestPlanEmitsRatingUpdateOnChange(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, true, false, []string{"atm"})}}
	vUser := vController(1000001, "home", []string{"ATM"}, "ZAU")
	vUser.Rating = 5

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm"}, fixedNow)

	assertOp(t, ops, OpUpdateRating)
}

func TestPlanEmitsMembershipWhenNotMember(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, false, false, []string{"atm"})}}
	vUser := vController(1000001, "home", []string{"ATM"}, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm"}, fixedNow)

	assertOp(t, ops, OpUpdateMembership)
}

func TestPlanEmitsVisitUpdateWhenStatusChanges(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, true, true, nil)}}
	vUser := vController(1000001, "home", nil, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm"}, fixedNow)

	assertOp(t, ops, OpUpdateVisit)
}

func TestPlanEmitsVisitUpdateWhenVisitorHomeFacilityChanges(t *testing.T) {
	z := zUser(1000001, true, true, nil)
	z.HomeFacility = "BOS"
	roster := zau.Roster{Home: []zau.User{z}}
	vUser := vController(1000001, "visit", nil, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm"}, fixedNow)

	assertOp(t, ops, OpUpdateVisit)
}

func TestPlanEmitsRoleUpdateWithUnion(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, true, false, []string{"atm"})}}
	vUser := vController(1000001, "home", []string{"ATM", "EC"}, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm", "ec"}, fixedNow)

	if len(ops) != 1 || ops[0].Kind != OpUpdateRoles {
		t.Fatalf("expected 1 role update op, got %+v", ops)
	}

	want := []string{"atm", "ec"}
	if !slices.Equal(ops[0].Roles, want) {
		t.Errorf("expected roles %v, got %v", want, ops[0].Roles)
	}
}

func TestPlanNoRoleUpdateWhenNoNewRoles(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, true, false, []string{"atm"})}}
	vUser := vController(1000001, "home", []string{"ATM"}, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"atm"}, fixedNow)

	assertNoOp(t, ops, OpUpdateRoles)
}

func TestPlanEmitsRemoveMemberForMissingCID(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, true, false, []string{"atm"})}}

	ops := Plan(roster, nil, []string{"atm"}, fixedNow)

	assertOp(t, ops, OpRemoveMember)
}

func TestPlanEmitsRemoveCertsAfterSixMonths(t *testing.T) {
	removalDate := fixedNow.AddDate(0, -7, 0)
	z := zUser(1000001, false, false, nil)
	z.RemovalDate = &removalDate
	z.CertCodes = []string{"C1"}
	roster := zau.Roster{Removed: []zau.User{z}}

	ops := Plan(roster, nil, []string{"atm"}, fixedNow)

	assertOp(t, ops, OpRemoveCerts)
}

func TestPlanNoRemoveCertsWithinSixMonths(t *testing.T) {
	removalDate := fixedNow.AddDate(0, -1, 0)
	z := zUser(1000001, false, false, nil)
	z.RemovalDate = &removalDate
	z.CertCodes = []string{"C1"}
	roster := zau.Roster{Removed: []zau.User{z}}

	ops := Plan(roster, nil, []string{"atm"}, fixedNow)

	assertNoOp(t, ops, OpRemoveCerts)
}

func TestPlanNoRemoveCertsWithoutCertCodes(t *testing.T) {
	removalDate := fixedNow.AddDate(0, -7, 0)
	z := zUser(1000001, false, false, nil)
	z.RemovalDate = &removalDate
	roster := zau.Roster{Removed: []zau.User{z}}

	ops := Plan(roster, nil, []string{"atm"}, fixedNow)

	assertNoOp(t, ops, OpRemoveCerts)
}

func TestPlanHandlesEmptyHomeRoster(t *testing.T) {
	z := zUser(1000002, true, true, []string{"ec"})
	z.HomeFacility = "BOS"
	roster := zau.Roster{Home: nil, Visiting: []zau.User{z}}
	vUser := vController(1000002, "visit", []string{"EC"}, "ZAU")

	ops := Plan(roster, []vatusa.Controller{vUser}, []string{"ec"}, fixedNow)

	if len(ops) != 1 || ops[0].Kind != OpUpdateVisit {
		t.Fatalf("expected 1 visit update op, got %+v", ops)
	}
}

func TestPlanIsDeterministic(t *testing.T) {
	roster := zau.Roster{Home: []zau.User{zUser(1000001, true, false, []string{"atm"})}}
	vUser := vController(1000001, "home", []string{"ATM", "EC"}, "ZAU")

	first := Plan(roster, []vatusa.Controller{vUser}, []string{"atm", "ec"}, fixedNow)
	second := Plan(roster, []vatusa.Controller{vUser}, []string{"atm", "ec"}, fixedNow)

	if !reflect.DeepEqual(first, second) {
		t.Errorf("expected deterministic plans:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

func TestPlanDoesNotMutateInputRoster(t *testing.T) {
	home := []zau.User{zUser(1000001, true, false, []string{"atm"})}
	roster := zau.Roster{Home: home}
	vUser := vController(1000001, "home", []string{"ATM"}, "ZAU")

	_ = Plan(roster, []vatusa.Controller{vUser}, []string{"atm"}, fixedNow)

	if len(home) != 1 {
		t.Fatalf("input home slice modified, len=%d", len(home))
	}

	if len(home[0].RoleCodes) != 1 || home[0].RoleCodes[0] != "atm" {
		t.Fatalf("input home user modified: %+v", home[0])
	}
}
