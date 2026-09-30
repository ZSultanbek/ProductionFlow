package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// errorResponse is the only error shape this API returns. Every failure, from a
// 400 to a panic, goes out through writeError so the client has exactly one
// error format to parse.
type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeJSON marshals before writing the header, so an encoding failure produces
// a clean 500 instead of a 200 with a truncated body.
func writeJSON(w http.ResponseWriter, status int, body any) {
	buf, err := json.Marshal(body)
	if err != nil {
		slog.Error("encode response body", "error", err)
		writeRaw(w, http.StatusInternalServerError,
			[]byte(`{"error":{"code":"internal","message":"internal server error"}}`))
		return
	}
	writeRaw(w, status, buf)
}

// writeError sends the error envelope. code is the machine-readable string the
// client switches on (day_not_found, not_implemented); message is for humans.
// Never pass a raw Go error message here — see CLAUDE.md.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorBody{Code: code, Message: message}})
}

func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
