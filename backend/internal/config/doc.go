// Package config reads process configuration from the environment.
//
// Every value has a default that works for `make dev` against the Postgres in
// docker-compose.yml, so a fresh clone runs without an .env file. Nothing here
// reads flags or files — the environment is the single source.
package config
