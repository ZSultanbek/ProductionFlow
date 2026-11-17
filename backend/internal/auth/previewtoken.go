package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Errors from Previews.Verify. Apply answers 409 preview_stale for the first
// (the token is not one we would have issued for this request) and 409
// preview_expired for the second.
var (
	ErrPreviewInvalid = errors.New("preview token: invalid")
	ErrPreviewExpired = errors.New("preview token: expired")
)

// PreviewClaims is what a preview token remembers about the preview it came
// from (docs/architecture.md §4.4). Apply checks each one against the request
// and against the database before it writes anything.
type PreviewClaims struct {
	Kind  string `json:"k"`
	DayID string `json:"d"`
	// MoveHash is domain.DayMove.Fingerprint of the requested "to".
	MoveHash string `json:"m"`
	// DayUpdatedAt is the day's updated_at in Unix microseconds — Postgres
	// precision, so the comparison in apply is exact.
	DayUpdatedAt int64 `json:"u"`
	// ImpactHash is domain.Impact.Fingerprint of what the producer was shown.
	ImpactHash string `json:"i"`
	UserID     string `json:"s"`
	ExpiresAt  int64  `json:"e"` // Unix seconds
}

// Previews signs and checks preview tokens. Stateless on purpose: no table to
// store previews in, no job to clean them up.
//
// The token is base64url(payload) "." base64url(HMAC-SHA256(key, payload)).
// key is derived from the JWT secret with a fixed label, so there is still only
// one secret to manage, and a preview token can never pass as an access token
// or the other way round.
type Previews struct {
	key []byte
	ttl time.Duration
	now func() time.Time
}

func NewPreviews(jwtSecret []byte, ttl time.Duration) *Previews {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte("productionflow/preview-token/v1"))
	return &Previews{key: mac.Sum(nil), ttl: ttl, now: time.Now}
}

// Sign fills in the expiry and returns the token and when it expires.
func (p *Previews) Sign(c PreviewClaims) (string, time.Time) {
	expiresAt := p.now().UTC().Truncate(time.Second).Add(p.ttl)
	c.ExpiresAt = expiresAt.Unix()

	payload, _ := json.Marshal(c) // a struct of strings and ints cannot fail
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(p.sign(encoded)), expiresAt
}

// Verify checks the signature first and the expiry second: an expired token
// with a bad signature is invalid, not expired.
func (p *Previews) Verify(token string) (PreviewClaims, error) {
	encoded, sig, ok := strings.Cut(token, ".")
	if !ok {
		return PreviewClaims{}, ErrPreviewInvalid
	}
	gotSig, err := base64.RawURLEncoding.DecodeString(sig)
	// hmac.Equal takes the same time whatever the input, so the response time
	// does not reveal how many bytes of a forged signature were right.
	if err != nil || !hmac.Equal(gotSig, p.sign(encoded)) {
		return PreviewClaims{}, ErrPreviewInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return PreviewClaims{}, ErrPreviewInvalid
	}
	var c PreviewClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return PreviewClaims{}, ErrPreviewInvalid
	}
	if !p.now().Before(time.Unix(c.ExpiresAt, 0)) {
		return PreviewClaims{}, ErrPreviewExpired
	}
	return c, nil
}

func (p *Previews) sign(encodedPayload string) []byte {
	mac := hmac.New(sha256.New, p.key)
	mac.Write([]byte(encodedPayload))
	return mac.Sum(nil)
}
