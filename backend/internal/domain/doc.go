// Package domain holds the entities and the business rules that act on them.
//
// What belongs here:
//   - Entity structs: Project, ShootingDay, Scene, Participant, ChangeEvent.
//   - Pure functions over already-loaded data. The impact computation lives
//     here: given a shooting day plus its scenes and participants, return what
//     a change affects. It takes values, not a database handle, so it is unit
//     testable without Postgres.
//   - Validation rules that are true regardless of transport or storage
//     (a call time must exist on a date, a role must be one of the three).
//
// What does not belong here:
//   - SQL, pgx types, or anything that knows a database exists.
//   - net/http types, JSON tags, or request/response shapes.
//
// If a function in this package needs to load something to do its job, that is
// a sign the loading belongs in store and the loaded values should be passed in.
package domain
