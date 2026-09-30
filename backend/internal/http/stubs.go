package http

import "net/http"

// notImplemented is the placeholder for every route that has a contract in
// docs/api.md but no implementation yet. It exists so the Flutter client can
// point at a real server and get a parseable envelope instead of a connection
// error. Replace these one endpoint at a time; delete this file when the last
// one is gone.
func notImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented",
		"this endpoint is not implemented yet")
}

// handleNotFound keeps unknown paths in the same error envelope as everything
// else. Note the trade-off: because this is registered on "/", a known path with
// the wrong method falls through here as 404 rather than the 405 ServeMux would
// return on its own.
func handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
}
