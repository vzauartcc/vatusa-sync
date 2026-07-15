package zau

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	ErrMissingEnv    = errors.New("missing environment variables")
	ErrRequestFailed = errors.New("request failed")
)

func SendData(ctx context.Context, method, path string, cid int, data any) error {
	var payload []byte

	var err error

	if data == nil {
		payload = []byte("{}")
	} else {
		payload, err = json.Marshal(data)
		if err != nil {
			log.Printf("Error marshalling %s payload for CID %d: %v\n", method, cid, err)

			return err
		}
	}

	bodyReader := bytes.NewReader(payload)

	req, err := NewZauAuthRequest(ctx, method, path, bodyReader)
	if err != nil {
		log.Printf("failed to create %s request for CID %d: %v\n", method, cid, err)

		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("%s request failed for CID %d: %v\n", method, cid, err)

		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		log.Printf("%s request failed for CID %d with status %s: %s\n", method, cid, resp.Status, responseBody)

		return fmt.Errorf("status %s: %s: %w", resp.Status, responseBody, ErrRequestFailed)
	}

	return nil
}

func FetchData(ctx context.Context) Roster {
	req, err := NewZauAuthRequest(ctx, http.MethodGet, "/user", nil)
	if err != nil {
		log.Printf("Error creating request for ZAU Controller data: %v\n", err)
		return Roster{}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error fetching ZAU Controller data: %v\n", err)
		return Roster{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Roster{}
	}

	var zauData Roster

	err = json.NewDecoder(resp.Body).Decode(&zauData)
	if err != nil {
		log.Printf("Error unmarshaling ZAU Controller data: %v\n", err)
		return Roster{}
	}

	return zauData
}

func FetchRoles(ctx context.Context) []string {
	req, err := NewZauAuthRequest(ctx, http.MethodGet, "/controller/role", nil)
	if err != nil {
		log.Printf("Error creating ZAU Role request: %v\n", err)
		return []string{}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error fetching ZAU Role data: %v\n", err)
		return []string{}
	}
	defer resp.Body.Close()

	var zauData []Role

	err = json.NewDecoder(resp.Body).Decode(&zauData)
	if err != nil {
		log.Printf("Error unmarshaling ZAU Role data: %v\n", err)
		return []string{}
	}

	availableRoles := make([]string, 0)
	for _, role := range zauData {
		availableRoles = append(availableRoles, role.Code)
	}

	return availableRoles
}

func NewZauAuthRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	baseURL := strings.TrimSpace(os.Getenv("ZAU_API_URL"))
	if baseURL == "" {
		return nil, ErrMissingEnv
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	req, err := http.NewRequestWithContext(ctx, method, os.Getenv("ZAU_API_URL")+path, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv("ZAU_API_KEY")))

	if method != http.MethodGet && method != http.MethodHead && req.Header.Get("Content-Type") == "" {
		// Set Content-Type for non-GET requests
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	return req, nil
}
