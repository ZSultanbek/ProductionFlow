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

// isUUID reports whether s is written like a UUID: 8-4-4-4-12 hex digits.
// Path ids do not need this — a malformed one simply matches no row — but a
// value that is going to be cast in SQL (a cursor's id) must be checked first,
// or a tampered cursor turns into a database error and a 500.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}
