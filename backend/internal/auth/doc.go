// Package auth handles identity: password hashing and access tokens.
//
// What belongs here:
//   - Password hashing and verification — bcrypt, cost 10 (password.go).
//   - Issuing and verifying the access token — HS256 JWT carrying only the
//     user id, 7-day lifetime, no refresh flow (jwt.go). See CLAUDE.md and
//     docs/architecture.md §4.3.
//   - Later: signing and verifying impact preview tokens (§4.4).
//
// What does not belong here:
//   - Roles and permissions. A role belongs to a participant in a project, not
//     to the user, so "is this person a producer here" is answered per request
//     from the database, never from the token.
//   - Loading anything from Postgres. This package is pure crypto helpers and
//     is tested without a database.
//   - The HTTP middleware that reads the Authorization header — that lives in
//     internal/http, because it writes error responses.
package auth
