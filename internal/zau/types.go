package zau

import (
	"time"
)

type Roster struct {
	Home     []User `json:"home"`
	Visiting []User `json:"visiting"`
	Removed  []User `json:"removed"`
}

type Role struct {
	Name  string `json:"name"`
	Code  string `json:"code"`
	Order int    `json:"order"`
	Class string `json:"class"`
}

type User struct {
	CID                  int        `json:"cid"`
	FName                string     `json:"fname"`
	LName                string     `json:"lname"`
	Email                string     `json:"email"`
	Rating               int        `json:"rating"`
	HomeFacility         string     `json:"homeFacility"`
	OI                   *string    `json:"oi,omitempty"`
	FlagBroadcastOptedIn bool       `json:"broadcast"`
	IsMember             bool       `json:"member"`
	IsVisitor            bool       `json:"vis"`
	JoinDate             *time.Time `json:"joinDate,omitempty"`
	RemovalDate          *time.Time `json:"removalDate,omitempty"`
	UseNamePrivacy       bool       `json:"prefName"`
	CertCodes            []string   `json:"certCodes"`
	RoleCodes            []string   `json:"roleCodes"`
}

type PostControllerPayload struct {
	// API Payload DTOs.
	CID              int       `json:"cid"`
	FName            string    `json:"fname"`
	LName            string    `json:"lname"`
	Rating           int       `json:"rating"`
	HomeFacility     string    `json:"home"`
	Email            string    `json:"email"`
	BroadcastOptedIn bool      `json:"broadcast"`
	IsMember         bool      `json:"member"`
	IsVisitor        bool      `json:"vis"`
	RoleCodes        []string  `json:"roleCodes"`
	CreatedAt        time.Time `json:"createdAt"`
	JoinDate         time.Time `json:"joinDate"`
	UseNamePrivacy   bool      `json:"prefName"`
}

type PatchControllerPayload struct {
	FName            string `json:"fname"`
	LName            string `json:"lname"`
	Email            string `json:"email"`
	BroadcastOptedIn bool   `json:"broadcast"`
	UseNamePrivacy   bool   `json:"prefName"`
}

type MemberControllerPayload struct {
	IsMember bool      `json:"member"`
	JoinDate time.Time `json:"joinDate"`
}

type VisitControllerPayload struct {
	IsVisitor    bool   `json:"vis"`
	HomeFacility string `json:"homeFacility"`
}

type RatingControllerPayload struct {
	Rating int `json:"rating"`
}

type RolesControllerPayload struct {
	Roles []string `json:"roles"`
}
