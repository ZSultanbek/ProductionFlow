// Package store is the only package that talks to Postgres.
//
// What belongs here:
//   - The pgx pool and its lifecycle (see store.go).
//   - Queries, one file per aggregate: project.go, shooting_day.go, scene.go,
//     participant.go, change_event.go. Each file holds the SQL for that
//     aggregate and nothing else.
//   - Mapping rows to domain structs, and transactions that span aggregates
//     (applying a change writes the entity and its change_event together).
//
// What does not belong here:
//   - Business rules. A query loads data; deciding what a change affects is
//     domain's job. Resist computing the impact view in SQL — it is easier to
//     test and easier to read as a pure function.
//   - HTTP status codes or error envelopes. Return plain errors and let the
//     http package translate them.
package store
