package git

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Compile-time check that ConnectionValidator implements port.GitConnectionValidator.
var _ port.GitConnectionValidator = (*ConnectionValidator)(nil)

const (
	defaultGitHubAPIBase = "https://api.github.com"
	probeTimeout         = 5 * time.Second

	headerOAuthScopes    = "X-OAuth-Scopes"
	headerTokenExpiry    = "github-authentication-token-expiration"
	tokenPrefixClassic   = "ghp_"
	tokenPrefixFineGrain = "github_pat_"
)

// ConnectionValidator probes a PAT against GitHub (or a self-hosted Gitea) to read
// the account login, granted scopes (classic only), and expiry. It pins the GitHub
// API base to operator config and sends the token only to the configured/derived
// host (security §5: probe-target pinning).
type ConnectionValidator struct {
	githubBase string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewGitHubConnectionValidator builds the validator. githubBase defaults to the
// public GitHub API when empty (supports GHE via operator config).
func NewGitHubConnectionValidator(githubBase string, logger *slog.Logger) *ConnectionValidator {
	base := strings.TrimRight(strings.TrimSpace(githubBase), "/")
	if base == "" {
		base = defaultGitHubAPIBase
	}
	return &ConnectionValidator{
		githubBase: base,
		httpClient: &http.Client{Timeout: probeTimeout},
		logger:     logger,
	}
}

// Probe validates token against provider. For github, baseURL is ignored (the
// validator's pinned base is used). For gitea, baseURL must be the project's host.
func (v *ConnectionValidator) Probe(ctx context.Context, provider, baseURL, token string) (port.ProbeResult, error) {
	switch provider {
	case "", "github":
		return v.probeGitHub(ctx, token)
	case "gitea":
		return v.probeGitea(ctx, baseURL, token)
	default:
		// Unknown provider: refuse to send the token anywhere.
		return port.ProbeResult{}, &port.ProbeError{Kind: port.ProbeNetwork, Message: "unsupported provider"}
	}
}

func (v *ConnectionValidator) probeGitHub(ctx context.Context, token string) (port.ProbeResult, error) {
	resp, err := v.do(ctx, v.githubBase+"/user", token)
	if err != nil {
		return port.ProbeResult{}, err
	}
	if pe := classifyHTTPStatus(resp.statusCode); pe != nil {
		return port.ProbeResult{}, pe
	}

	result := port.ProbeResult{
		Login:     extractLogin(resp.body),
		TokenType: ClassifyTokenType(token),
	}
	if scopes := parseScopes(resp.header.Get(headerOAuthScopes)); len(scopes) > 0 {
		result.Scopes = scopes
	}
	if exp := parseTokenExpiry(resp.header.Get(headerTokenExpiry)); exp != nil {
		result.ExpiresAt = exp
	}
	return result, nil
}

func (v *ConnectionValidator) probeGitea(ctx context.Context, baseURL, token string) (port.ProbeResult, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return port.ProbeResult{}, &port.ProbeError{Kind: port.ProbeNetwork, Message: "no gitea host configured"}
	}
	resp, err := v.do(ctx, base+"/api/v1/user", token)
	if err != nil {
		return port.ProbeResult{}, err
	}
	if pe := classifyHTTPStatus(resp.statusCode); pe != nil {
		return port.ProbeResult{}, pe
	}
	return port.ProbeResult{
		Login:     extractLogin(resp.body),
		TokenType: ClassifyTokenType(token),
	}, nil
}

// probeResponse is the minimal, body-closed slice of an HTTP probe we need.
type probeResponse struct {
	statusCode int
	header     http.Header
	body       []byte
}

// do issues the GET, fully reads+closes a bounded slice of the body, and returns the
// status/header/body. It returns a *port.ProbeError (classified, sanitized — never
// carrying the token) on transport failure.
func (v *ConnectionValidator) do(ctx context.Context, endpoint, token string) (*probeResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &port.ProbeError{Kind: port.ProbeNetwork, Message: "could not build probe request"}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, classifyTransportError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Bound the read; we only need login + headers.
	body := readAtMost(resp.Body, 8192)
	return &probeResponse{statusCode: resp.StatusCode, header: resp.Header, body: body}, nil
}

func readAtMost(r io.Reader, limit int) []byte {
	buf := make([]byte, limit)
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil || n == 0 {
			break
		}
	}
	return buf[:total]
}

// classifyHTTPStatus maps an HTTP status to a typed, sanitized ProbeError, or nil on 2xx.
func classifyHTTPStatus(code int) *port.ProbeError {
	switch {
	case code >= 200 && code < 300:
		return nil
	case code == http.StatusUnauthorized:
		return &port.ProbeError{Kind: port.ProbeUnauthorized, Message: "provider rejected the token"}
	case code == http.StatusForbidden:
		return &port.ProbeError{Kind: port.ProbeForbidden, Message: "token lacks required permission"}
	case code == http.StatusTooManyRequests:
		return &port.ProbeError{Kind: port.ProbeRateLimited, Message: "provider rate limited the probe"}
	case code >= 500:
		return &port.ProbeError{Kind: port.Probe5xx, Message: "provider returned a server error"}
	default:
		return &port.ProbeError{Kind: port.ProbeNetwork, Message: "unexpected provider response"}
	}
}

// classifyTransportError maps a transport error to a typed ProbeError. The original
// error is NOT embedded (it can carry a credential-bearing URL); only a fixed,
// sanitized message is surfaced.
func classifyTransportError(err error) *port.ProbeError {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &port.ProbeError{Kind: port.ProbeDNS, Message: "could not resolve provider host"}
	}
	var tlsErr *tls.CertificateVerificationError
	if errors.As(err, &tlsErr) {
		return &port.ProbeError{Kind: port.ProbeTLS, Message: "TLS verification failed"}
	}
	if msg := strings.ToLower(err.Error()); strings.Contains(msg, "tls") || strings.Contains(msg, "certificate") || strings.Contains(msg, "x509") {
		return &port.ProbeError{Kind: port.ProbeTLS, Message: "TLS verification failed"}
	}
	return &port.ProbeError{Kind: port.ProbeNetwork, Message: "could not reach provider"}
}

// ClassifyTokenType classifies a PAT from its prefix: ghp_ -> classic,
// github_pat_ -> fine_grained, anything else -> unknown.
func ClassifyTokenType(token string) string {
	switch {
	case strings.HasPrefix(token, tokenPrefixFineGrain):
		return "fine_grained"
	case strings.HasPrefix(token, tokenPrefixClassic):
		return "classic"
	default:
		return "unknown"
	}
}

// parseScopes splits the comma-separated X-OAuth-Scopes header into trimmed,
// non-empty scope strings.
func parseScopes(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// parseTokenExpiry parses the github-authentication-token-expiration header, which
// GitHub emits in a few formats. Returns nil when absent or unparseable.
func parseTokenExpiry(header string) *time.Time {
	h := strings.TrimSpace(header)
	if h == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 -0700 MST",
		time.RFC3339,
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, h); err == nil {
			return &t
		}
	}
	return nil
}

// extractLogin pulls the "login" field from a /user JSON body without a full
// struct (kept tolerant of provider field differences). Returns "" if absent.
func extractLogin(body []byte) string {
	return jsonStringField(string(body), "login")
}

// jsonStringField does a minimal, dependency-free scan for "key":"value". It is
// used only for the advisory login display, so a best-effort match is sufficient.
func jsonStringField(s, key string) string {
	needle := "\"" + key + "\""
	i := strings.Index(s, needle)
	if i < 0 {
		return ""
	}
	rest := s[i+len(needle):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return ""
	}
	return rest[:end]
}
