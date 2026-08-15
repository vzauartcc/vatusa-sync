package sync

import (
	"strings"
	"time"

	"github.com/vzauartcc/roster-sync/internal/vatusa"
	"github.com/vzauartcc/roster-sync/internal/zau"
)

// Plan computes the operations needed to bring the ZAU roster in line with
// the VATUSA roster. It is pure: no I/O, no logging, and it does not mutate
// its inputs.
func Plan(
	zauRoster zau.Roster,
	vatusaControllers []vatusa.Controller,
	availableRoles []string,
	now time.Time,
) []Operation {
	allZauControllers := make([]zau.User, 0,
		len(zauRoster.Home)+len(zauRoster.Visiting)+len(zauRoster.Removed))
	allZauControllers = append(allZauControllers, zauRoster.Home...)
	allZauControllers = append(allZauControllers, zauRoster.Visiting...)
	allZauControllers = append(allZauControllers, zauRoster.Removed...)

	zauByCID := make(map[int]zau.User, len(allZauControllers))
	for _, controller := range allZauControllers {
		if _, exists := zauByCID[controller.CID]; !exists {
			zauByCID[controller.CID] = controller
		}
	}

	availableRoleSet := make(map[string]bool, len(availableRoles))
	for _, role := range availableRoles {
		availableRoleSet[strings.ToLower(role)] = true
	}

	ops := make([]Operation, 0, len(vatusaControllers))

	sixMonthsAgo := now.AddDate(0, -6, -1)

	for _, controller := range zauRoster.Removed {
		if controller.RemovalDate == nil {
			continue
		}

		isOlderThanSixMonths := controller.RemovalDate.Before(sixMonthsAgo)

		hasCertCodes := len(controller.CertCodes) > 0

		if isOlderThanSixMonths && hasCertCodes {
			ops = append(ops, Operation{
				Kind: OpRemoveCerts,
				CID:  controller.CID,
			})
		}
	}

	vatusaCIDs := make(map[int]bool, len(vatusaControllers))
	for _, controller := range vatusaControllers {
		vatusaCIDs[controller.CID] = true
	}

	for _, controller := range allZauControllers {
		if !controller.IsMember {
			continue
		}

		if _, exists := vatusaCIDs[controller.CID]; !exists {
			ops = append(ops, Operation{
				Kind: OpRemoveMember,
				CID:  controller.CID,
			})
		}
	}

	for _, vUser := range vatusaControllers {
		zUser, exists := zauByCID[vUser.CID]

		desiredRoles := filterRoles(vUser.Roles, availableRoleSet)

		if !exists {
			ops = append(ops, Operation{
				Kind:  OpCreate,
				CID:   vUser.CID,
				VUser: vUser,
				Roles: desiredRoles,
			})

			continue
		}

		if checkCoreInfo(vUser, zUser) {
			ops = append(ops, Operation{
				Kind:  OpUpdateCore,
				CID:   vUser.CID,
				VUser: vUser,
			})
		}

		if checkVisitingStatus(vUser, zUser, vUser.Membership != homeMembership) {
			ops = append(ops, Operation{
				Kind:  OpUpdateVisit,
				CID:   vUser.CID,
				VUser: vUser,
			})
		}

		if !zUser.IsMember {
			ops = append(ops, Operation{
				Kind:  OpUpdateMembership,
				CID:   vUser.CID,
				VUser: vUser,
			})
		}

		if zUser.Rating != vUser.Rating {
			ops = append(ops, Operation{
				Kind:  OpUpdateRating,
				CID:   vUser.CID,
				VUser: vUser,
			})
		}

		roles := addNewRoles(zUser.RoleCodes, desiredRoles)
		if len(roles) != len(zUser.RoleCodes) {
			ops = append(ops, Operation{
				Kind:  OpUpdateRoles,
				CID:   vUser.CID,
				ZUser: zUser,
				Roles: roles,
			})
		}
	}

	return ops
}

// filterRoles returns the lowercase roles assignable on the ZAU roster:
// roles for ZAU or ZHQ that exist in the roster's available roles.
func filterRoles(roles []vatusa.Role, available map[string]bool) []string {
	assignable := make([]string, 0, len(roles))

	for _, userRole := range roles {
		if userRole.Facility != "ZAU" && userRole.Facility != "ZHQ" {
			continue
		}

		lcRole := strings.ToLower(userRole.Role)

		if available[lcRole] {
			assignable = append(assignable, lcRole)
		}
	}

	return assignable
}

// addNewRoles appends roles to the current set, preserving existing roles and
// never removing any.
func addNewRoles(current []string, newRoles []string) []string {
	seen := make(map[string]bool, len(current)+len(newRoles))

	merged := make([]string, 0, len(current)+len(newRoles))
	merged = append(merged, current...)

	for _, role := range current {
		seen[role] = true
	}

	for _, role := range newRoles {
		if seen[role] {
			continue
		}

		seen[role] = true
		merged = append(merged, role)
	}

	return merged
}

func checkCoreInfo(vUser vatusa.Controller, zUser zau.User) bool {
	return vUser.FName != zUser.FName || vUser.LName != zUser.LName || vUser.Email != zUser.Email ||
		vUser.BroadcastOptedIn != zUser.FlagBroadcastOptedIn ||
		vUser.NamePrivacyEnabled != zUser.UseNamePrivacy
}

func checkVisitingStatus(vUser vatusa.Controller, zUser zau.User, visiting bool) bool {
	return zUser.IsVisitor != visiting || (visiting && zUser.HomeFacility != vUser.Facility)
}
