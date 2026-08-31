package sync

import (
	"github.com/vzauartcc/roster-sync/internal/vatusa"
	"github.com/vzauartcc/roster-sync/internal/zau"
)

type OperationKind int

const (
	OpCreate OperationKind = iota
	OpUpdateCore
	OpUpdateMembership
	OpUpdateVisit
	OpUpdateRating
	OpUpdateRoles
	OpRemoveMember
	OpRemoveCerts
	OpAddACE
)

func (k OperationKind) String() string {
	switch k {
	case OpCreate:
		return "create"
	case OpUpdateCore:
		return "update-core"
	case OpUpdateMembership:
		return "update-membership"
	case OpUpdateVisit:
		return "update-visit"
	case OpUpdateRating:
		return "update-rating"
	case OpUpdateRoles:
		return "update-roles"
	case OpRemoveMember:
		return "remove-member"
	case OpRemoveCerts:
		return "remove-certs"
	case OpAddACE:
		return "add-ace"
	default:
		return "unknown"
	}
}

// Operation describes a single change to apply to the ZAU roster.
// Each operation carries the data its executor call needs.
type Operation struct {
	Kind OperationKind
	CID  int

	VUser vatusa.Controller
	ZUser zau.User
	Roles []string
}

// Result holds the counters reported after a run.
type Result struct {
	Added         int
	MadeMember    int
	MadeVisitor   int
	UpdatedCore   int
	UpdatedRating int
	UpdatedRoles  int
	RemovedMember int
	CertsRemoved  int
	ACEGrants     int
}
