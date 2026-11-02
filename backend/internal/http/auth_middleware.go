package http

import (
	"context"
	"net/http"
	"strings"
)

type contextKey int

const userIDKey contextKey = iota

// requireAuth lets a request through only with a valid access token in
// `Authorization: Bearer <token>`, and puts the user id on the context for the
// handler. Every failure is the same 401: the client's only correct reaction is
// to drop the token and show the login screen, whatever the reason.
func requireAuth(deps Deps, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeUnauthorized(w)
			return
		}

		userID, err := deps.Tokens.Verify(token)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
	}
}

// userIDFrom returns the id requireAuth stored. Only call it from a handler that
// is wrapped in requireAuth; anywhere else it returns "".
func userIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// bearerToken extracts the token from an Authorization header value. The scheme
// name is case-insensitive (RFC 9110), so "bearer x" is accepted too.
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

func writeUnauthorized(w http.ResponseWriter) {
	// Tells generic HTTP clients which auth scheme this API expects.
	w.Header().Set("WWW-Authenticate", `Bearer realm="productionflow"`)
	writeError(w, http.StatusUnauthorized, "unauthorized", "log in again")
}
