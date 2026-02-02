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

type Certification struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	Order    int    `json:"order"`
	Class    string `json:"class"`
	Facility string `json:"facility"`
}

type Absence struct {
	Controller     int       `json:"controller"`
	ExpirationDate time.Time `json:"expiration_date"`
	Deleted        bool      `json:"deleted"`
}

type DiscordInfo struct {
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expires      time.Time `json:"expires"`
}

type User struct {
	CID                  int          `json:"cid"`
	FName                string       `json:"fname"`
	LName                string       `json:"lname"`
	Email                string       `json:"email"`
	Rating               int          `json:"rating"`
	OI                   *string      `json:"oi,omitempty"` // Nullable OI field
	FlagBroadcastOptedIn bool         `json:"broadcast"`
	IsMember             bool         `json:"member"`
	IsVisitor            bool         `json:"vis"`
	HomeFacility         string       `json:"home_facility,omitempty"` // Optional Home Facility field
	Bio                  string       `json:"bio"`
	Avatar               *string      `json:"avatar,omitempty"`       // Nullable Avatar field
	JoinDate             *time.Time   `json:"join_date,omitempty"`    // Optional Join Date field
	RemovalDate          *time.Time   `json:"removal_date,omitempty"` // Optional Removal Date field
	UseNamePrivacy       bool         `json:"pref_name"`
	DiscordInfo          *DiscordInfo `json:"discord_info,omitempty"` // Nullable Discord Info struct
	DiscordID            string       `json:"discord,omitempty"`
	IDSToken             string       `json:"ids_token,omitempty"`
	CertCodes            []string     `json:"cert_codes"`
	RoleCodes            []string     `json:"role_codes"`
	TrainingMilestones   []string     `json:"training_milestone"`
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
	IsVisitor bool `json:"vis"`
}

type RatingControllerPayload struct {
	Rating int `json:"rating"`
}
