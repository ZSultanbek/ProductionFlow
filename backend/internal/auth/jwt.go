package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// issuer goes into every token and is required on the way back in, so a token
// minted by some other service that happens to share the secret is rejected.
const issuer = "productionflow"

// ErrInvalidToken covers every way a token can be unusable: malformed, wrong
// signature, wrong algorithm, wrong issuer, expired. The caller answers 401 for
// all of them and has no reason to tell them apart.
var ErrInvalidToken = errors.New("invalid token")

// Tokens issues and verifies access tokens. The token carries only the user id:
// roles belong to a participant in a project, not to the user, so they are
// looked up per request instead of being frozen into a token for a week.
type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokens builds a Tokens. secret must be the same across restarts or every
// issued token stops working; config.Load makes sure it is set in production.
func NewTokens(secret []byte, ttl time.Duration) *Tokens {
	return &Tokens{secret: secret, ttl: ttl, now: time.Now}
}

// Issue returns a signed HS256 token for userID and the moment it expires.
func (t *Tokens) Issue(userID string) (string, time.Time, error) {
	now := t.now().UTC().Truncate(time.Second)
	expiresAt := now.Add(t.ttl)

	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// Verify checks the signature, algorithm, issuer and expiry, and returns the
// user id the token was issued for.
func (t *Tokens) Verify(token string) (string, error) {
	var claims jwt.RegisteredClaims

	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		// Pinning the algorithm is what stops the classic "alg: none" and
		// HS/RS confusion attacks. Never remove this option.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
