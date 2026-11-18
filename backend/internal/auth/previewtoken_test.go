package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testSecret = "test-secret-test-secret-test-secret"

func previewsAt(secret string, at *time.Time) *Previews {
	p := NewPreviews([]byte(secret), 10*time.Minute)
	p.now = func() time.Time { return *at }
	return p
}

var sampleClaims = PreviewClaims{
	Kind: "move_shooting_day", DayID: "day-1", MoveHash: "m", DayUpdatedAt: 1762070400123456,
	ImpactHash: "i", UserID: "user-1",
}

func TestPreviewTokenRoundTrip(t *testing.T) {
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	p := previewsAt(testSecret, &now)

	token, expiresAt := p.Sign(sampleClaims)
	if !expiresAt.Equal(now.Add(10 * time.Minute)) {
		t.Errorf("expiresAt = %v, want ten minutes from now", expiresAt)
	}

	got, err := p.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	want := sampleClaims
	want.ExpiresAt = expiresAt.Unix()
	if got != want {
		t.Errorf("claims = %+v, want %+v", got, want)
	}
}

func TestPreviewTokenExpires(t *testing.T) {
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	p := previewsAt(testSecret, &now)
	token, _ := p.Sign(sampleClaims)

	now = now.Add(10 * time.Minute)
	if _, err := p.Verify(token); !errors.Is(err, ErrPreviewExpired) {
		t.Errorf("err = %v, want ErrPreviewExpired at exactly ten minutes", err)
	}
}

func TestPreviewTokenRejectsTampering(t *testing.T) {
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	p := previewsAt(testSecret, &now)
	token, _ := p.Sign(sampleClaims)

	// Someone edits the payload to point the preview at another day.
	other := sampleClaims
	other.DayID = "day-2"
	forged, _ := p.Sign(other)
	payloadOfForged, _, _ := strings.Cut(forged, ".")
	_, sigOfReal, _ := strings.Cut(token, ".")

	for name, bad := range map[string]string{
		"swapped payload":  payloadOfForged + "." + sigOfReal,
		"no signature":     payloadOfForged,
		"empty":            "",
		"garbage":          "not.a-token",
		"other secret":     mustSign(t, "another-secret-another-secret-xx", now),
		"access token-ish": "eyJhbGciOiJIUzI1NiJ9.e30.c2ln",
	} {
		if _, err := p.Verify(bad); !errors.Is(err, ErrPreviewInvalid) {
			t.Errorf("%s: err = %v, want ErrPreviewInvalid", name, err)
		}
	}
}

func mustSign(t *testing.T, secret string, now time.Time) string {
	t.Helper()
	token, _ := previewsAt(secret, &now).Sign(sampleClaims)
	return token
}
