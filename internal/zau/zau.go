package zau

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cristalhq/jwt/v5"
)

var (
	ErrMissingEnv    = errors.New("missing environment variables")
	ErrRequestFailed = errors.New("request failed")
)

type Client struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		baseURL: strings.TrimSpace(baseURL),
		apiKey:  strings.TrimSpace(apiKey),
		hc:      httpClient,
	}
}

func (c *Client) SetHomeController(
	ctx context.Context,
	cid int,
	member bool,
	joinDate time.Time,
) error {
	slog.Info("setting home status", "cid", cid, "rostered", member)

	return c.sendData(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/controller/%d/member", cid),
		cid,
		MemberControllerPayload{
			JoinDate: joinDate,
			IsMember: member,
		},
	)
}

func (c *Client) SetVisitingController(
	ctx context.Context,
	cid int,
	visiting bool,
	homeFacility string,
) error {
	slog.Info("setting visiting status", "cid", cid, "visiting", visiting)

	return c.sendData(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/controller/%d/visit", cid),
		cid,
		VisitControllerPayload{
			HomeFacility: homeFacility,
			IsVisitor:    visiting,
		},
	)
}

func (c *Client) SetRating(ctx context.Context, cid int, rating int) error {
	slog.Info("setting rating", "cid", cid, "rating", rating)

	return c.sendData(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/controller/%d/rating", cid),
		cid,
		RatingControllerPayload{
			Rating: rating,
		},
	)
}

func (c *Client) SetCoreDetails(
	ctx context.Context,
	cid int,
	fname string,
	lname string,
	email string,
	allowEmails bool,
	namePrivacy bool,
) error {
	slog.Info(
		"setting core details",
		"cid",
		cid,
		"fname",
		fname,
		"lname",
		lname,
		"email",
		email,
		"broadcast_opted_in",
		allowEmails,
		"name_privacy",
		namePrivacy,
	)

	return c.sendData(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/controller/%d", cid),
		cid,
		PatchControllerPayload{
			FName:            fname,
			LName:            lname,
			Email:            email,
			UseNamePrivacy:   namePrivacy,
			BroadcastOptedIn: allowEmails,
		},
	)
}

func (c *Client) SetRoles(ctx context.Context, cid int, roles []string) error {
	slog.Info("setting roles", "cid", cid, "roles", roles)

	return c.sendData(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/controller/%d/roles", cid),
		cid,
		RolesControllerPayload{
			Roles: roles,
		},
	)
}

func (c *Client) CreateUser(
	ctx context.Context,
	cid int,
	fname string,
	lname string,
	email string,
	allowEmails bool,
	namePrivacy bool,
	rating int,
	homeFacility string,
	member bool,
	visiting bool,
	joinDate time.Time,
	roles []string,
) error {
	slog.Info(
		"setting home status",
		"cid",
		cid,
		"fname",
		fname,
		"lname",
		lname,
		"home_controller",
		member,
		"visiting",
		visiting,
		"rating",
		rating,
	)

	return c.sendData(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/controller/%d", cid),
		cid,
		PostControllerPayload{
			CID:              cid,
			FName:            fname,
			LName:            lname,
			Email:            email,
			UseNamePrivacy:   namePrivacy,
			BroadcastOptedIn: allowEmails,
			Rating:           rating,
			HomeFacility:     homeFacility,
			IsMember:         member,
			IsVisitor:        visiting,
			JoinDate:         joinDate,
			RoleCodes:        roles,
		},
	)
}

func (c *Client) RemoveCerts(ctx context.Context, cid int) error {
	slog.Info("removing certs", "cid", cid)

	return c.sendData(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/controller/%d/remove-cert", cid),
		cid,
		nil,
	)
}

func (c *Client) FetchRoster(ctx context.Context) (Roster, error) {
	req, err := c.newZauAuthRequest(ctx, http.MethodGet, "/user", nil)
	if err != nil {
		return Roster{}, err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return Roster{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Roster{}, fmt.Errorf("status %s: %w", resp.Status, ErrRequestFailed)
	}

	var zauData Roster

	err = json.NewDecoder(resp.Body).Decode(&zauData)
	if err != nil {
		return Roster{}, err
	}

	return zauData, nil
}

func (c *Client) FetchRoles(ctx context.Context) ([]string, error) {
	req, err := c.newZauAuthRequest(ctx, http.MethodGet, "/controller/role", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %w", resp.Status, ErrRequestFailed)
	}

	var zauData []Role

	err = json.NewDecoder(resp.Body).Decode(&zauData)
	if err != nil {
		return nil, err
	}

	availableRoles := make([]string, 0, len(zauData))
	for _, role := range zauData {
		availableRoles = append(availableRoles, role.Code)
	}

	return availableRoles, nil
}

func (c *Client) sendData(ctx context.Context, method, path string, cid int, data any) error {
	var payload []byte

	var err error

	if data == nil {
		payload = []byte("{}")
	} else {
		payload, err = json.Marshal(data)
		if err != nil {
			slog.Error("error marshalling payload", "method", method, "cid", cid, "error", err)

			return err
		}
	}

	bodyReader := bytes.NewReader(payload)

	req, err := c.newZauAuthRequest(ctx, method, path, bodyReader)
	if err != nil {
		slog.Error("failed to create request", "method", method, "cid", cid, "error", err)

		return err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		slog.Error(
			"request failed",
			"method", method,
			"path", req.URL.Path,
			"cid", cid,
			"error", err,
		)

		return err
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		slog.Error(
			"request failed with non-success status",
			"method", method,
			"path", req.URL.Path,
			"cid", cid,
			"status", resp.Status,
			"response", string(responseBody),
		)

		return fmt.Errorf("status %s: %s: %w", resp.Status, responseBody, ErrRequestFailed)
	}

	return nil
}

func (c *Client) newZauAuthRequest(
	ctx context.Context,
	method, path string,
	body io.Reader,
) (*http.Request, error) {
	if c.baseURL == "" {
		return nil, ErrMissingEnv
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.generateJWT())

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "roster-sync/1.0")
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *Client) generateJWT() string {
	key := []byte(c.apiKey)

	signer, err := jwt.NewSignerHS(jwt.HS256, key)
	if err != nil {
		slog.Error("error generating jwt", "error", err)

		return ""
	}

	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		Subject:   "roster-sync",
	}

	builder := jwt.NewBuilder(signer)

	token, err := builder.Build(claims)
	if err != nil {
		slog.Error("error signing jwt", "error", err)

		return ""
	}

	return token.String()
}
