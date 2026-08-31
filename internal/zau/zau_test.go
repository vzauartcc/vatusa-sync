package zau

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cristalhq/jwt/v5"
)

const (
	testATMCode = "atm"
	testCIDPath = "/controller/1234567"
)

func TestNewClientDefaultsToTimeout(t *testing.T) {
	client := NewClient("https://zau.example.com", "key", nil)

	if client.hc.Timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", client.hc.Timeout)
	}
}

func TestFetchRosterDecodesAndAuthenticates(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}

			if req.URL.Path != "/user" {
				t.Errorf("expected path /user, got %s", req.URL.Path)
			}

			auth := req.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				t.Fatalf("expected Bearer token, got %q", auth)
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			verifier, err := jwt.NewVerifierHS(jwt.HS256, []byte("test-key"))
			if err != nil {
				t.Fatalf("failed to create verifier: %v", err)
			}

			_, err = jwt.Parse([]byte(tokenStr), verifier)
			if err != nil {
				t.Fatalf("invalid or unverifiable JWT: %v", err)
			}

			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(
				[]byte(
					`{"home":[{"cid":1234567,"fname":"John","lname":"Doe","email":"john@example.com","rating":3,"homeFacility":"ZAU","broadcast":true,"member":true,"vis":false,"prefName":false,"certCodes":[],"roleCodes":["atm"]}],"visiting":[],"removed":[]}`,
				),
			)
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	roster, err := client.FetchRoster(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(roster.Home) != 1 {
		t.Fatalf("expected 1 home controller, got %d", len(roster.Home))
	}

	if roster.Home[0].CID != 1234567 || roster.Home[0].RoleCodes[0] != testATMCode {
		t.Errorf("unexpected roster: %+v", roster.Home[0])
	}
}

func TestFetchRosterReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	_, err := client.FetchRoster(context.Background())
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestFetchRolesReturnsCodes(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/controller/role" {
				t.Errorf("expected path /controller/role, got %s", req.URL.Path)
			}

			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(
				[]byte(
					`[{"name":"ATM","code":"ATM","order":1,"class":"staff"},{"name":"EC","code":"EC","order":2,"class":"staff"}]`,
				),
			)
		}),
	)
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

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			rawBody, _ := io.ReadAll(req.Body)

			gotMethod = req.Method
			gotPath = req.URL.Path
			gotBody = string(rawBody)
			gotContentType = req.Header.Get("Content-Type")

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetRoles(context.Background(), 1234567, []string{testATMCode, "ec"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPut || gotPath != testCIDPath {
		t.Errorf("expected PUT %s, got %s %s", testCIDPath, gotMethod, gotPath)
	}

	if gotContentType != "application/json" {
		t.Errorf("expected application/json content type, got %q", gotContentType)
	}

	var payload RolesControllerPayload

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("invalid body %q: %v", gotBody, err)
	}

	if len(payload.Roles) != 2 || payload.Roles[0] != testATMCode || payload.Roles[1] != "ec" {
		t.Errorf("unexpected roles payload: %+v", payload)
	}
}

func TestSetHomeControllerSendsPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			body, _ := io.ReadAll(req.Body)

			gotMethod = req.Method
			gotPath = req.URL.Path
			gotBody = string(body)

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetHomeController(
		context.Background(),
		1234567,
		true,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	)
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

func TestSetVisitingControllerSendsPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			body, _ := io.ReadAll(req.Body)

			gotMethod = req.Method
			gotPath = req.URL.Path
			gotBody = string(body)

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetVisitingController(context.Background(), 1234567, true, "BOS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/controller/1234567/visit" {
		t.Errorf("expected PATCH /controller/1234567/visit, got %s %s", gotMethod, gotPath)
	}

	var payload VisitControllerPayload

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("invalid body %q: %v", gotBody, err)
	}

	if !payload.IsVisitor || payload.HomeFacility != "BOS" {
		t.Errorf("unexpected payload: %+v", payload)
	}
}

func TestSetRatingSendsPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			body, _ := io.ReadAll(req.Body)

			gotMethod = req.Method
			gotPath = req.URL.Path
			gotBody = string(body)

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetRating(context.Background(), 1234567, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/controller/1234567/rating" {
		t.Errorf("expected PATCH /controller/1234567/rating, got %s %s", gotMethod, gotPath)
	}

	var payload RatingControllerPayload

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("invalid body %q: %v", gotBody, err)
	}

	if payload.Rating != 5 {
		t.Errorf("unexpected payload: %+v", payload)
	}
}

func TestSetCoreDetailsSendsPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			body, _ := io.ReadAll(req.Body)

			gotMethod = req.Method
			gotPath = req.URL.Path
			gotBody = string(body)

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetCoreDetails(
		context.Background(),
		1234567,
		"Jane",
		"Smith",
		"jane@example.com",
		true,
		false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != testCIDPath {
		t.Errorf("expected PATCH %s, got %s %s", testCIDPath, gotMethod, gotPath)
	}

	var payload PatchControllerPayload

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("invalid body %q: %v", gotBody, err)
	}

	if payload.FName != "Jane" || payload.LName != "Smith" || payload.Email != "jane@example.com" ||
		!payload.BroadcastOptedIn || payload.UseNamePrivacy {
		t.Errorf("unexpected payload: %+v", payload)
	}
}

func TestCreateUserSendsPost(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			body, _ := io.ReadAll(req.Body)

			gotMethod = req.Method
			gotPath = req.URL.Path
			gotBody = string(body)

			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	joinDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	err := client.CreateUser(
		context.Background(),
		1234567,
		"Jane",
		"Smith",
		"jane@example.com",
		true,
		false,
		3,
		"ZAU",
		true,
		false,
		joinDate,
		[]string{testATMCode},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != testCIDPath {
		t.Errorf("expected POST %s, got %s %s", testCIDPath, gotMethod, gotPath)
	}

	var payload PostControllerPayload

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("invalid body %q: %v", gotBody, err)
	}

	if payload.CID != 1234567 || !payload.IsMember || payload.IsVisitor ||
		payload.FName != "Jane" || payload.Rating != 3 ||
		payload.HomeFacility != "ZAU" || payload.JoinDate != joinDate {
		t.Errorf("unexpected payload: %+v", payload)
	}

	if len(payload.RoleCodes) != 1 || payload.RoleCodes[0] != testATMCode {
		t.Errorf("unexpected roles payload: %+v", payload.RoleCodes)
	}
}

func TestRemoveCertsSendsPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			body, _ := io.ReadAll(req.Body)

			gotMethod = req.Method
			gotPath = req.URL.Path
			gotBody = string(body)

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.RemoveCerts(context.Background(), 1234567)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/controller/1234567/remove-cert" {
		t.Errorf("expected PATCH /controller/1234567/remove-cert, got %s %s", gotMethod, gotPath)
	}

	if gotBody != "{}" {
		t.Errorf("expected empty object body, got %q", gotBody)
	}
}

func TestSendDataReturnsErrorOnNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(`{"error":"nope"}`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	err := client.SetRating(context.Background(), 1234567, 3)
	if err == nil {
		t.Fatal("expected error on non-success status")
	}

	if !errors.Is(err, ErrRequestFailed) {
		t.Fatalf("expected ErrRequestFailed, got %v", err)
	}
}

func TestFetchRosterReturnsErrorWhenMissingBaseURL(t *testing.T) {
	client := NewClient("", "test-key", nil)

	_, err := client.FetchRoster(context.Background())
	if !errors.Is(err, ErrMissingEnv) {
		t.Fatalf("expected ErrMissingEnv, got %v", err)
	}
}

func TestFetchRolesReturnsErrorWhenMissingBaseURL(t *testing.T) {
	client := NewClient("", "test-key", nil)

	_, err := client.FetchRoles(context.Background())
	if !errors.Is(err, ErrMissingEnv) {
		t.Fatalf("expected ErrMissingEnv, got %v", err)
	}
}

func TestFetchRosterReturnsErrorOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`not json`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	_, err := client.FetchRoster(context.Background())
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestFetchRolesReturnsErrorOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`not json`))
		}),
	)
	defer server.Close()

	client := NewClient(server.URL, "test-key", nil)

	_, err := client.FetchRoles(context.Background())
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
