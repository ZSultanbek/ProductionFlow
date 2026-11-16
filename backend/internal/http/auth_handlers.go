package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/auth"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// UserStore is the slice of store.DB the auth handlers need. An interface so
// the handlers can be tested with an in-memory fake instead of Postgres.
type UserStore interface {
	CreateUser(ctx context.Context, email, passwordHash, displayName string) (domain.User, error)
	UserByEmail(ctx context.Context, email string) (domain.User, string, error)
	UserByID(ctx context.Context, id string) (domain.User, error)
}

// --- wire shapes (docs/api.md, "Auth — US-01") ---

type userJSON struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

func toUserJSON(u domain.User) userJSON {
	return userJSON{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   formatInstant(u.CreatedAt),
	}
}

type authResponse struct {
	Token     string   `json:"token"`
	ExpiresAt string   `json:"expires_at"`
	User      userJSON `json:"user"`
}

type meResponse struct {
	User userJSON `json:"user"`
}

// --- handlers ---

// handleRegister: POST /api/auth/register. Creates the account and logs the
// user straight in, so the client does not need a second round trip.
func handleRegister(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email       string `json:"email"`
			Password    string `json:"password"`
			DisplayName string `json:"display_name"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}

		reg, fieldErrs := domain.ValidateRegistration(req.Email, req.Password, req.DisplayName)
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		hash, err := auth.HashPassword(reg.Password)
		if err != nil {
			writeInternalError(w, deps.Logger, "register: hash password", err)
			return
		}

		user, err := deps.Users.CreateUser(r.Context(), reg.Email, hash, reg.DisplayName)
		if errors.Is(err, store.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email_taken", "an account with this email already exists")
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "register: create user", err)
			return
		}

		writeAuthResponse(w, deps, http.StatusCreated, user)
	}
}

// handleLogin: POST /api/auth/login. Unknown email and wrong password give the
// same response in the same time, so the endpoint cannot be used to discover
// which emails have accounts.
func handleLogin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}

		email, fieldErrs := domain.ValidateLogin(req.Email, req.Password)
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		user, hash, err := deps.Users.UserByEmail(r.Context(), email)
		if errors.Is(err, store.ErrNotFound) {
			auth.BurnPasswordCheck(req.Password)
			writeInvalidCredentials(w)
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "login: load user", err)
			return
		}

		if !auth.CheckPassword(hash, req.Password) {
			writeInvalidCredentials(w)
			return
		}

		writeAuthResponse(w, deps, http.StatusOK, user)
	}
}

// handleMe: GET /api/me. The client calls this at startup to find out whether
// its stored token still works.
func handleMe(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := deps.Users.UserByID(r.Context(), userIDFrom(r.Context()))
		if errors.Is(err, store.ErrNotFound) {
			// A valid token for an account that no longer exists.
			writeUnauthorized(w)
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "me: load user", err)
			return
		}

		writeJSON(w, http.StatusOK, meResponse{User: toUserJSON(user)})
	}
}

func writeAuthResponse(w http.ResponseWriter, deps Deps, status int, user domain.User) {
	token, expiresAt, err := deps.Tokens.Issue(user.ID)
	if err != nil {
		writeInternalError(w, deps.Logger, "issue token", err)
		return
	}
	writeJSON(w, status, authResponse{
		Token:     token,
		ExpiresAt: formatInstant(expiresAt),
		User:      toUserJSON(user),
	})
}

func writeInvalidCredentials(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "invalid_credentials", "wrong email or password")
}
