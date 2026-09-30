// Package http is the transport layer: routes, handlers, middleware and the
// JSON shapes that go over the wire.
//
// The package is deliberately named http and shadows net/http from the caller's
// side, so cmd/api imports it as apihttp. Inside this package, http refers to
// the standard library as usual.
//
// What belongs here:
//   - Route registration (router.go) using http.ServeMux method patterns.
//   - Handlers: decode the request, call domain or store, encode the response.
//     A handler should read as a short list of steps with no rules of its own.
//   - Request/response DTOs with snake_case JSON tags, kept separate from the
//     domain structs so the wire format can change independently.
//   - Middleware: request logging, panic recovery, auth.
//
// What does not belong here:
//   - Business rules. The impact computation in particular must not live in a
//     handler (CLAUDE.md is explicit about this).
//   - SQL.
//
// Errors always go out through writeError so every failure has the same
// envelope: {"error": {"code": "...", "message": "..."}}.
package http
