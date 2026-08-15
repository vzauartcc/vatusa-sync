package zau

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientDefaultsToTimeout(t *testing.T) {
	client := NewClient("https://zau.example.com", "key", nil)

	if client.hc.Timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", client.hc.Timeout)
	}
}

func TestFetchRosterDecodesAndAuthenticates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}

		if req.URL.Path != "/user" {
			t.Errorf("expected path /user, got %s", req.URL.Path)
		}

		if got := req.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %q", got)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"home":[{"cid":1234567,"fname":"John","lname":"Doe","email":"john@example.com","rating":3,"homeFacility":"ZAU","broadcast":true,"member":true,"vis":false,"prefName":false,"certCodes":[],"roleCodes":["atm"]}],"visiting":[],"removed":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	roster, err := client.FetchRoster(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(roster.Home) != 1 {
		t.Fatalf("expected 1 home controller, got %d", len(roster.Home))
	}

	if roster.Home[0].CID != 1234567 || roster.Home[0].RoleCodes[0] != "atm" {
		t.Errorf("unexpected roster: %+v", roster.Home[0])
	}
}

func TestFetchRosterReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	_, err := client.FetchRoster(context.Background())
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestFetchRolesReturnsCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/controller/role" {
			t.Errorf("expected path /controller/role, got %s", req.URL.Path)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[{"name":"ATM","code":"ATM","order":1,"class":"staff"},{"name":"EC","code":"EC","order":2,"class":"staff"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	roles, err := client.FetchRoles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(roles) != 2 || roles[0] != "ATM" || roles[1] != "EC" {
		t.Errorf("unexpected roles: %v", roles)
	}
}

func TestSetRolesSendsPutWithRoles(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		rawBody, _ := io.ReadAll(req.Body)

		gotMethod = req.Method
		gotPath = req.URL.Path
		gotBody = string(rawBody)
		gotContentType = req.Header.Get("Content-Type")

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetRoles(context.Background(), 1234567, []string{"atm", "ec"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPut || gotPath != "/controller/1234567" {
		t.Errorf("expected PUT /controller/1234567, got %s %s", gotMethod, gotPath)
	}

	if gotContentType != "application/json" {
		t.Errorf("expected application/json content type, got %q", gotContentType)
	}

	var payload RolesControllerPayload

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("invalid body %q: %v", gotBody, err)
	}

	if len(payload.Roles) != 2 || payload.Roles[0] != "atm" || payload.Roles[1] != "ec" {
		t.Errorf("unexpected roles payload: %+v", payload)
	}
}

func TestSetHomeControllerSendsPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)

		gotMethod = req.Method
		gotPath = req.URL.Path
		gotBody = string(body)

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetHomeController(context.Background(), 1234567, true, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/controller/1234567/member" {
		t.Errorf("expected PATCH /controller/1234567/member, got %s %s", gotMethod, gotPath)
	}

	var payload MemberControllerPayload

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("invalid body %q: %v", gotBody, err)
	}

	if !payload.IsMember || payload.JoinDate.Year() != 2026 {
		t.Errorf("unexpected payload: %+v", payload)
	}
}
