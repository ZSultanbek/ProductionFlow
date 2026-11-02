package http

import (
	"encoding/json"
	"net/http"
	"time"
)

// maxBodyBytes caps every JSON request body. Nothing in the API needs more than
// a few hundred bytes; the cap stops a client from making the server buffer an
// arbitrarily large upload.
const maxBodyBytes = 1 << 20 // 1 MiB

// decodeJSON reads the request body into dst. On failure it has already written
// a 422 and returns false — the handler just returns.
//
// Unknown fields are ignored rather than rejected, so an older server keeps
// working with a newer client that sends something extra.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeValidationError(w, map[string]string{"body": "must be a JSON object matching the API contract"})
		return false
	}
	return true
}

// formatInstant renders a timestamp the way docs/api.md promises: RFC 3339 in
// UTC with a Z, to the second. time.Time's own JSON encoding would keep the
// server's offset and nanoseconds.
func formatInstant(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
