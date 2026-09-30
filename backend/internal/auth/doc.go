// Package auth handles identity: password hashing and JWT access tokens.
//
// What belongs here:
//   - Password hashing and verification (bcrypt or argon2id — pick one and
//     keep it; the choice needs a line in the PR since it is a new dependency).
//   - Issuing and verifying the access token. `Authorization: Bearer <jwt>`,
//     access token only — the MVP has no refresh flow, see CLAUDE.md.
//   - The claims struct (participant id, project id, role) and the middleware
//     that puts the authenticated participant on the request context.
//
// What does not belong here:
//   - Permission rules about domain entities ("can this role move a shooting
//     day"). That is a business rule and belongs in domain; auth only answers
//     who the caller is.
//   - Loading participants from Postgres. Take a lookup function or interface
//     so this package stays testable without a database.
//
// Nothing is implemented yet — the skeleton serves unauthenticated stubs.
package auth
