package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/vzauartcc/roster-sync/internal/vatusa"
	"github.com/vzauartcc/roster-sync/internal/zau"
)

func main() {
	if os.Getenv("ZAU_API_URL") == "" || os.Getenv("ZAU_API_KEY") == "" || os.Getenv("VATUSA_API_KEY") == "" {
		panic("Missing at least one environment variable. Check to make sure the following are set: 'VATUSA_API_KEY', 'ZAU_API_KEY', 'ZAU_API_URL'.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := cron.New(cron.WithChain(cron.Recover(cron.DefaultLogger)))

	err := setupScheduler(ctx, runner, doRosterSync)
	if err != nil {
		panic(err)
	}

	runner.Start()

	go doRosterSync(ctx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	<-sigs

	log.Println("Shutting down. . . .")

	stopCtx := runner.Stop()

	<-stopCtx.Done()

	log.Println("Bye!")
}

func setupScheduler(ctx context.Context, runner *cron.Cron, job func(context.Context)) error {
	_, err := runner.AddFunc("*/10 * * * *", func() {
		go job(ctx)
	})

	return err
}

func doRosterSync(ctx context.Context) {
	log.Println("\n⏳ Starting sync . . .")

	start := time.Now()

	zauControllers, vatusaControllers, failed := getControllers(ctx)
	if failed {
		return
	}

	log.Printf("Got %d controllers from ZAU\n", len(zauControllers.Home)+len(zauControllers.Visiting))

	allZauControllers, zauCertRemovalCIDs, makeNonMember := generateRosterSyncSlices(zauControllers, vatusaControllers)

	added := 0
	madeMember := 0
	madeVisitor := 0
	updatedCore := 0
	updatedRating := 0

	for _, vUser := range vatusaControllers {
		var zUser *zau.User

		for i := range allZauControllers {
			z := &allZauControllers[i]

			if z.CID == vUser.CID {
				zUser = z
				break
			}
		}

		isVisitor := vUser.Membership != "home"

		if zUser == nil {
			// New User
			log.Printf("Creating new user for %s %s (%d)\n", vUser.FName, vUser.LName, vUser.CID)

			err := createNewUser(ctx, vUser, isVisitor)
			if err == nil {
				added++
			}

			continue
		}

		// Existing User, update as necessary
		core, membership, vis, rating := updateExistingUser(ctx, *zUser, vUser, isVisitor)

		if core {
			updatedCore++
		}

		if membership {
			madeMember++
		}

		if vis {
			madeVisitor++
		}

		if rating {
			updatedRating++
		}
	}

	removedMember := removeMembers(ctx, makeNonMember)

	certsRemoved := removeCerts(ctx, zauCertRemovalCIDs)

	log.Printf("⌛ Done! Finished in %.1f seconds\n", time.Since(start).Seconds())

	if added > 0 {
		log.Printf("Added %d new users.\n", added)
	}

	if madeMember > 0 {
		log.Printf("Made %d users into members.\n", madeMember)
	}

	if madeVisitor > 0 {
		log.Printf("Changed %d users' visiting status.\n", madeVisitor)
	}

	if updatedCore > 0 {
		log.Printf("Updated %d users' core data.\n", updatedCore)
	}

	if updatedRating > 0 {
		log.Printf("Updated %d users' ratings.\n", updatedRating)
	}

	if removedMember > 0 {
		log.Printf("Removed %d users from the roster.\n", removedMember)
	}

	if certsRemoved > 0 {
		log.Printf("Removed %d expired certs for non-member users.\n\n", certsRemoved)
	}
}

func getControllers(ctx context.Context) (zau.Roster, []vatusa.Controller, bool) {
	vatusaControllers, err := vatusa.FetchData(ctx)

	if err != nil || len(vatusaControllers) == 0 {
		log.Printf("Failed to fetch VATUSA controllers")

		return zau.Roster{}, nil, true
	}

	log.Printf("Got %d controllers from VATUSA\n", len(vatusaControllers))

	zauControllers := zau.FetchData(ctx)
	if len(zauControllers.Home) == 0 {
		log.Printf("Failed to fetch ZAU controllers")

		return zau.Roster{}, nil, true
	}

	return zauControllers, vatusaControllers, false
}

func generateRosterSyncSlices(zauControllers zau.Roster, vatusaControllers []vatusa.Controller) ([]zau.User, []int, []int) {
	sixMonthsAgo := time.Now().AddDate(0, -6, -1)
	zauMemberCIDs := make([]int, 0)

	allZauControllers := zauControllers.Home
	allZauControllers = append(allZauControllers, zauControllers.Visiting...)
	allZauControllers = append(allZauControllers, zauControllers.Removed...)

	for _, controller := range allZauControllers {
		if controller.IsMember {
			zauMemberCIDs = append(zauMemberCIDs, controller.CID)
		}
	}

	zauCertRemovalCIDs := make([]int, 0)

	for _, controller := range zauControllers.Removed {
		if controller.RemovalDate == nil {
			continue // Skip if removalDate is missing
		}

		isOlderThanSixMonths := controller.RemovalDate.Before(sixMonthsAgo)

		hasCertCodes := len(controller.CertCodes) > 0

		if isOlderThanSixMonths && hasCertCodes {
			zauCertRemovalCIDs = append(zauCertRemovalCIDs, controller.CID)
		}
	}

	vatusaAllCIDs := make([]int, 0)

	for _, controller := range vatusaControllers {
		vatusaAllCIDs = append(vatusaAllCIDs, controller.CID)
	}

	makeNonMember := make([]int, 0)

	vatusaCidMap := make(map[int]bool, len(vatusaAllCIDs))
	for _, cid := range vatusaAllCIDs {
		vatusaCidMap[cid] = true
	}

	for _, cid := range zauMemberCIDs {
		if _, exists := vatusaCidMap[cid]; !exists {
			makeNonMember = append(makeNonMember, cid)
		}
	}

	return allZauControllers, zauCertRemovalCIDs, makeNonMember
}
func createNewUser(ctx context.Context, vUser vatusa.Controller, isVisitor bool) error {
	availableRoles := zau.FetchRoles(ctx)
	// Create list of available roles
	availableRolesMap := make(map[string]bool, len(availableRoles))
	for _, role := range availableRoles {
		availableRolesMap[strings.ToLower(role)] = true
	}

	assignableRoles := make([]string, 0, len(vUser.Roles))

	for _, userRole := range vUser.Roles {
		lcRole := strings.ToLower(userRole.Role)

		if availableRolesMap[lcRole] {
			assignableRoles = append(assignableRoles, lcRole)
		}
	}

	return zau.SendData(ctx, http.MethodPost, fmt.Sprintf("/controller/%d", vUser.CID), vUser.CID, zau.PostControllerPayload{
		CID:              vUser.CID,
		FName:            vUser.FName,
		LName:            vUser.LName,
		Rating:           vUser.Rating,
		HomeFacility:     vUser.Facility,
		Email:            vUser.Email,
		BroadcastOptedIn: vUser.BroadcastOptedIn,
		IsMember:         true,
		IsVisitor:        isVisitor,
		RoleCodes: func() []string {
			if !isVisitor {
				return assignableRoles
			}

			return []string{}
		}(),
		JoinDate:       vUser.FacilityJoinDate,
		UseNamePrivacy: vUser.NamePrivacyEnabled,
	})
}

func updateExistingUser(ctx context.Context, zUser zau.User, vUser vatusa.Controller, isVisitor bool) (bool, bool, bool, bool) {
	coreInfo := false
	membership := false
	visit := false
	rating := false

	// Update user if core info changed
	if vUser.FName != zUser.FName || vUser.LName != zUser.LName || vUser.Email != zUser.Email || vUser.BroadcastOptedIn != zUser.FlagBroadcastOptedIn || vUser.NamePrivacyEnabled != zUser.UseNamePrivacy {
		log.Printf("Updating user core info for %d", zUser.CID)

		err := zau.SendData(ctx, http.MethodPatch, fmt.Sprintf("/user/%d", vUser.CID), vUser.CID, zau.PatchControllerPayload{
			FName:            vUser.FName,
			LName:            vUser.LName,
			Email:            vUser.Email,
			BroadcastOptedIn: vUser.BroadcastOptedIn,
			UseNamePrivacy:   vUser.NamePrivacyEnabled,
		})
		if err == nil {
			coreInfo = true
		}
	}

	// Update membership if necessary
	if !zUser.IsMember {
		log.Printf("Updating user membership for %d", vUser.CID)

		err := zau.SendData(ctx, http.MethodPatch, fmt.Sprintf("/controller/%d/member", zUser.CID), zUser.CID, zau.MemberControllerPayload{
			IsMember: true,
			JoinDate: vUser.FacilityJoinDate,
		})
		if err == nil {
			membership = true
		}
	}

	// Update visiting status if necessary
	if zUser.IsVisitor != isVisitor {
		log.Printf("Updating user visit status for %d to %t", zUser.CID, isVisitor)

		err := zau.SendData(ctx, http.MethodPatch, fmt.Sprintf("/controller/%d/visit", zUser.CID), zUser.CID, zau.VisitControllerPayload{
			IsVisitor: isVisitor,
		})
		if err == nil {
			visit = true
		}
	}

	// Update rating status if necessary
	if zUser.Rating != vUser.Rating {
		log.Printf("Updating user rating for %d to %d from %d", zUser.CID, vUser.Rating, zUser.Rating)

		err := zau.SendData(ctx, http.MethodPatch, fmt.Sprintf("/controller/%d/rating", zUser.CID), zUser.CID, zau.RatingControllerPayload{
			Rating: vUser.Rating,
		})
		if err == nil {
			rating = true
		}
	}

	return coreInfo, membership, visit, rating
}

func removeMembers(ctx context.Context, makeNonMember []int) int {
	if len(makeNonMember) == 0 {
		return 0
	}

	retval := 0

	for _, cid := range makeNonMember {
		log.Printf("Removing user %d from roster", cid)

		err := zau.SendData(ctx, http.MethodPatch, fmt.Sprintf("/controller/%d/member", cid), cid, zau.MemberControllerPayload{
			IsMember: false,
			JoinDate: time.Now(), // Join date is ignored when removing members
		})
		if err == nil {
			retval++
		}
	}

	return retval
}

func removeCerts(ctx context.Context, zauCertRemovalCIDs []int) int {
	if len(zauCertRemovalCIDs) == 0 {
		return 0
	}

	retval := 0

	for _, cid := range zauCertRemovalCIDs {
		log.Printf("Removing certs for %d (6 months since rostered)", cid)

		err := zau.SendData(ctx, http.MethodPatch, fmt.Sprintf("/controller/%d/remove-cert", cid), cid, nil)
		if err == nil {
			retval++
		}
	}

	return retval
}
