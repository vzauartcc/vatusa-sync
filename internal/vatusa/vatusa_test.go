package vatusa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientDefaultsToTimeout(t *testing.T) {
	client := NewClient("https://vatusa.example.com", "key", nil)

	if client.hc.Timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", client.hc.Timeout)
	}
}

func TestFetchRosterBuildsURLAndDecodes(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RawQuery

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"testing":false,"data":[{"cid":1234567,"fname":"John","lname":"Doe","email":"john@example.com","facility":"ZAU","rating":3,"membership":"home"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", nil)

	controllers, err := client.FetchRoster(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(gotPath, "apikey=secret") {
		t.Errorf("expected apikey=secret in query, got %q", gotPath)
	}

	if !strings.Contains(gotPath, "&t=") {
		t.Errorf("expected timestamp param in query, got %q", gotPath)
	}

	if len(controllers) != 1 || controllers[0].CID != 1234567 {
		t.Fatalf("unexpected controllers: %+v", controllers)
	}
}

func TestFetchRosterReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", nil)

	_, err := client.FetchRoster(context.Background())
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestFetchRosterReturnsErrorWhenMissingConfig(t *testing.T) {
	client := NewClient("", "", nil)

	_, err := client.FetchRoster(context.Background())
	if err == nil {
		t.Fatal("expected error when URL or API key is missing")
	}
}

func TestCertSyncDateUnmarshal(t *testing.T) {
	var date CertSyncDate

	err := date.UnmarshalJSON([]byte(`"2025-12-08 03:23:09"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := time.Time(date).String(); got != "2025-12-08 03:23:09 +0000 UTC" {
		t.Errorf("unexpected date: %s", got)
	}

	var nullDate CertSyncDate

	err = nullDate.UnmarshalJSON([]byte(`null`))
	if err != nil {
		t.Fatalf("unexpected error on null: %v", err)
	}
}
