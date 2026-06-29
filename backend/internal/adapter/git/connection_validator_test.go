package git

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

func newTestValidator(base string) *ConnectionValidator {
	return NewGitHubConnectionValidator(base, testLogger())
}

func TestClassifyTokenType(t *testing.T) {
	cases := map[string]string{
		"ghp_0123456789abcdef":   "classic",
		"github_pat_11ABCDE_xyz": "fine_grained",
		"gho_oauthtoken":         "unknown",
		"random":                 "unknown",
	}
	for tok, want := range cases {
		if got := ClassifyTokenType(tok); got != want {
			t.Errorf("ClassifyTokenType(%q)=%q want %q", tok, got, want)
		}
	}
}

func TestProbeGitHub_Classic_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set(headerOAuthScopes, "repo, read:project, read:org")
		w.Header().Set(headerTokenExpiry, "2099-01-02 15:04:05 UTC")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"login":"octocat","id":1}`))
	}))
	defer srv.Close()

	res, err := newTestValidator(srv.URL).Probe(context.Background(), "github", "", "ghp_classictoken1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Login != "octocat" {
		t.Errorf("login=%q", res.Login)
	}
	if res.TokenType != "classic" {
		t.Errorf("token_type=%q", res.TokenType)
	}
	if len(res.Scopes) != 3 {
		t.Errorf("scopes=%v", res.Scopes)
	}
	if res.ExpiresAt == nil {
		t.Error("expected parsed expiry")
	}
}

func TestProbeGitHub_FineGrained_NoScopes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"login":"fg-user"}`))
	}))
	defer srv.Close()

	res, err := newTestValidator(srv.URL).Probe(context.Background(), "github", "", "github_pat_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TokenType != "fine_grained" {
		t.Errorf("token_type=%q", res.TokenType)
	}
	if len(res.Scopes) != 0 {
		t.Errorf("fine-grained must not surface scopes, got %v", res.Scopes)
	}
}

func TestProbeGitHub_StatusClassification(t *testing.T) {
	cases := []struct {
		code int
		kind port.ProbeErrorKind
	}{
		{http.StatusUnauthorized, port.ProbeUnauthorized},
		{http.StatusForbidden, port.ProbeForbidden},
		{http.StatusTooManyRequests, port.ProbeRateLimited},
		{http.StatusInternalServerError, port.Probe5xx},
		{http.StatusBadGateway, port.Probe5xx},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.code)
			_, _ = w.Write([]byte(`{"message":"nope"}`))
		}))
		_, err := newTestValidator(srv.URL).Probe(context.Background(), "github", "", "ghp_x")
		srv.Close()

		var pe *port.ProbeError
		if !errors.As(err, &pe) {
			t.Fatalf("code %d: expected *ProbeError, got %v", tc.code, err)
		}
		if pe.Kind != tc.kind {
			t.Errorf("code %d: kind=%q want %q", tc.code, pe.Kind, tc.kind)
		}
	}
}

func TestProbeKindTransient(t *testing.T) {
	transient := []port.ProbeErrorKind{port.Probe5xx, port.ProbeRateLimited, port.ProbeDNS, port.ProbeTLS, port.ProbeNetwork}
	for _, k := range transient {
		if !k.Transient() {
			t.Errorf("%q should be transient", k)
		}
	}
	for _, k := range []port.ProbeErrorKind{port.ProbeUnauthorized, port.ProbeForbidden} {
		if k.Transient() {
			t.Errorf("%q must NOT be transient (definitive)", k)
		}
	}
}

func TestProbe_NoTokenLeakInError(t *testing.T) {
	const secret = "ghp_supersecretvalue000111222333"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Echo the token back in the body to try to provoke a leak.
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad credentials ` + secret + `"}`))
	}))
	defer srv.Close()

	_, err := newTestValidator(srv.URL).Probe(context.Background(), "github", "", secret)
	if err == nil {
		t.Fatal("expected error")
	}
	if containsToken(err.Error(), secret) {
		t.Fatalf("probe error leaked the token: %q", err.Error())
	}
}

func containsToken(s, tok string) bool {
	return len(tok) > 0 && (s == tok || (len(s) >= len(tok) && indexOf(s, tok) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
