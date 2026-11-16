package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct horse" || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("hash does not look like bcrypt: %q", hash)
	}
	if !CheckPassword(hash, "correct horse") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "correct horse ") {
		t.Error("wrong password accepted")
	}
	if CheckPassword("not a hash", "correct horse") {
		t.Error("malformed hash accepted")
	}
}

func TestHashPasswordRefusesOver72Bytes(t *testing.T) {
	// bcrypt would otherwise ignore everything after byte 72, so two different
	// long passwords would unlock the same account.
	if _, err := HashPassword(strings.Repeat("x", 73)); err == nil {
		t.Error("73-byte password was hashed")
	}
}

func TestBurnPasswordCheckDoesNotPanic(t *testing.T) {
	BurnPasswordCheck("anything")
	BurnPasswordCheck("anything else")
}

// fixedClock returns a Tokens whose clock is under the test's control.
func fixedClock(secret string, ttl time.Duration, at *time.Time) *Tokens {
	tok := NewTokens([]byte(secret), ttl)
	tok.now = func() time.Time { return *at }
	return tok
}

func TestTokenRoundTrip(t *testing.T) {
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	tok := fixedClock("test-secret-test-secret-test-secret", 7*24*time.Hour, &now)

	signed, expiresAt, err := tok.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	if want := now.Add(7 * 24 * time.Hour); !expiresAt.Equal(want) {
		t.Errorf("expiresAt = %v, want %v", expiresAt, want)
	}

	userID, err := tok.Verify(signed)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("userID = %q, want %q", userID, "user-123")
	}
}

func TestTokenExpires(t *testing.T) {
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	tok := fixedClock("test-secret-test-secret-test-secret", time.Hour, &now)

	signed, _, err := tok.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(59 * time.Minute)
	if _, err := tok.Verify(signed); err != nil {
		t.Errorf("rejected a token one minute before expiry: %v", err)
	}

	now = now.Add(2 * time.Minute)
	if _, err := tok.Verify(signed); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("accepted an expired token (err = %v)", err)
	}
}

func TestTokenRejectsOtherSecret(t *testing.T) {
	now := time.Now()
	ours := fixedClock("our-secret-our-secret-our-secret-0", time.Hour, &now)
	theirs := fixedClock("another-secret-another-secret-000", time.Hour, &now)

	signed, _, err := theirs.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ours.Verify(signed); !errors.Is(err, ErrInvalidToken) {
		t.Error("accepted a token signed with a different secret")
	}
}

func TestTokenRejectsAlgNone(t *testing.T) {
	now := time.Now()
	tok := fixedClock("test-secret-test-secret-test-secret", time.Hour, &now)

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   "user-123",
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	})
	forged, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tok.Verify(forged); !errors.Is(err, ErrInvalidToken) {
		t.Error(`accepted an "alg: none" token`)
	}
}

func TestTokenRejectsOtherIssuer(t *testing.T) {
	now := time.Now()
	secret := []byte("test-secret-test-secret-test-secret")
	tok := fixedClock(string(secret), time.Hour, &now)

	foreign, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "someone-else",
		Subject:   "user-123",
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tok.Verify(foreign); !errors.Is(err, ErrInvalidToken) {
		t.Error("accepted a token from another issuer")
	}
}

func TestTokenRejectsGarbage(t *testing.T) {
	tok := NewTokens([]byte("test-secret-test-secret-test-secret"), time.Hour)
	for _, s := range []string{"", "not.a.token", "a.b", "eyJhbGciOiJIUzI1NiJ9.e30.x"} {
		if _, err := tok.Verify(s); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("accepted %q", s)
		}
	}
}
