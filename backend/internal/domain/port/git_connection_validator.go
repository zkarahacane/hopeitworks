package port

import (
	"context"
	"time"
)

// ProbeResult is the outcome of a successful credential probe (whoami + scope/expiry
// metadata). Returned only when the provider accepted the token (HTTP 2xx).
type ProbeResult struct {
	Login     string     // authenticated account login
	Scopes    []string   // X-OAuth-Scopes (classic PAT); empty for fine-grained
	ExpiresAt *time.Time // github-authentication-token-expiration header, if present
	TokenType string     // classic | fine_grained | unknown (classified from prefix)
}

// ProbeErrorKind is a FIXED classification of a probe failure. It maps to a
// persisted validation_error code; raw upstream text is never surfaced (A3).
type ProbeErrorKind string

const (
	// ProbeUnauthorized is HTTP 401 — the provider rejected the token.
	ProbeUnauthorized ProbeErrorKind = "unauthorized"
	// ProbeForbidden is HTTP 403 — token lacks permission/scope for the probe.
	ProbeForbidden ProbeErrorKind = "insufficient_scope"
	// Probe5xx is a transient server error (HTTP 5xx).
	Probe5xx ProbeErrorKind = "probe_5xx"
	// ProbeRateLimited is HTTP 429 — transient.
	ProbeRateLimited ProbeErrorKind = "rate_limited"
	// ProbeDNS is a name-resolution failure.
	ProbeDNS ProbeErrorKind = "dns_error"
	// ProbeTLS is a TLS handshake/certificate failure.
	ProbeTLS ProbeErrorKind = "tls_error"
	// ProbeNetwork is any other transport failure.
	ProbeNetwork ProbeErrorKind = "network_error"
)

// Transient reports whether a probe failure is non-definitive (do NOT flip a good
// connection to invalid; on PUT this maps to 503 and never overwrites a good row).
func (k ProbeErrorKind) Transient() bool {
	switch k {
	case Probe5xx, ProbeRateLimited, ProbeDNS, ProbeTLS, ProbeNetwork:
		return true
	default:
		return false
	}
}

// ProbeError is the typed, sanitized error a validator returns on a failed probe.
// Message is already scrubbed and safe to surface/log.
type ProbeError struct {
	Kind    ProbeErrorKind
	Message string
}

func (e *ProbeError) Error() string {
	if e.Message == "" {
		return string(e.Kind)
	}
	return string(e.Kind) + ": " + e.Message
}

// GitConnectionValidator probes a credential against its provider to obtain the
// account login, granted scopes, and expiry. It sends the token ONLY to a trusted,
// operator/admin-configured host (probe-target pinning, security §5).
type GitConnectionValidator interface {
	// Probe validates token against provider. baseURL is required for self-hosted
	// providers (gitea) and ignored for github (the validator pins the GitHub API
	// base from operator config). On 2xx it returns a populated ProbeResult and a
	// nil error; otherwise it returns a *ProbeError.
	Probe(ctx context.Context, provider, baseURL, token string) (ProbeResult, error)
}
