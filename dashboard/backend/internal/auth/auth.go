// Package auth is a minimal bearer-token authenticator with two roles:
// "viewer" (read-only) and "operator" (read + control actions). It's
// intentionally not OIDC/JWT — this dashboard has one deployment target
// and a handful of operators, and static tokens issued out-of-band cover
// that without a second service to run.
package auth

import (
	"context"
	"net/http"
	"strings"
)

type Identity struct {
	Actor string
	Role  string // "viewer" or "operator"
}

func (id Identity) CanWrite() bool { return id.Role == "operator" }

type Authenticator struct {
	tokens map[string]Identity // bearer token -> identity
}

// New parses AUTH_TOKENS-style config: "token:actor:role,token2:actor2:role2".
// An empty raw string disables auth entirely (every request becomes an
// unauthenticated "operator", matching pre-Phase-6 behavior) so existing
// local/demo setups keep working without issuing tokens.
func New(raw string) *Authenticator {
	tokens := make(map[string]Identity)
	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) != 3 {
			continue
		}
		role := parts[2]
		if role != "operator" && role != "viewer" {
			continue
		}
		tokens[parts[0]] = Identity{Actor: parts[1], Role: role}
	}
	return &Authenticator{tokens: tokens}
}

func (a *Authenticator) Enabled() bool { return len(a.tokens) > 0 }

type ctxKey struct{}

// Middleware authenticates every request except health/metrics probes,
// which cluster infra hits without credentials. When auth is disabled
// (no tokens configured) it passes every request through as an
// unauthenticated operator. GET requests need only a valid token;
// mutating requests additionally need the operator role.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		if !a.Enabled() {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, Identity{Actor: "operator", Role: "operator"})))
			return
		}

		// EventSource (used for the SSE stream) can't set custom headers,
		// so it authenticates via ?token= instead of Authorization.
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		id, ok := a.tokens[token]
		if !ok || token == "" {
			http.Error(w, `{"error":"missing or invalid bearer token"}`, http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet && !id.CanWrite() {
			http.Error(w, `{"error":"viewer role cannot perform this action"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

// FromContext returns the caller's identity, or the zero Identity if
// none was attached (should only happen for the exempted health routes).
func FromContext(ctx context.Context) Identity {
	id, _ := ctx.Value(ctxKey{}).(Identity)
	return id
}
