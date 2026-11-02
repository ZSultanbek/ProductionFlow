package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

// listResponse is the envelope every list endpoint returns (docs/api.md,
// Conventions): {"data": [...], "next_cursor": "..." | null}.
type listResponse[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor"`
}

// cursorJSON is what an opaque cursor decodes to. Clients must not build or
// parse it — docs/api.md says so — which leaves us free to change it.
type cursorJSON struct {
	CreatedAt time.Time `json:"t"`
	ID        string    `json:"id"`
}

func encodeCursor(k store.Keyset) string {
	raw, _ := json.Marshal(cursorJSON{CreatedAt: k.CreatedAt, ID: k.ID})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(s string) (store.Keyset, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return store.Keyset{}, false
	}
	var c cursorJSON
	if err := json.Unmarshal(raw, &c); err != nil || c.ID == "" || c.CreatedAt.IsZero() {
		return store.Keyset{}, false
	}
	return store.Keyset{CreatedAt: c.CreatedAt, ID: c.ID}, true
}

// parsePage reads ?cursor= and ?limit=. after is nil for the first page.
func parsePage(r *http.Request) (after *store.Keyset, limit int, errs domain.FieldErrors) {
	q := r.URL.Query()
	errs = domain.FieldErrors{}
	limit = defaultPageSize

	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPageSize {
			errs["limit"] = "must be a whole number from 1 to 100"
		} else {
			limit = n
		}
	}

	if raw := q.Get("cursor"); raw != "" {
		k, ok := decodeCursor(raw)
		if !ok {
			errs["cursor"] = "is not a cursor this API issued"
		} else {
			after = &k
		}
	}

	return after, limit, errs
}

// page trims a result fetched with limit+1 rows down to limit, and returns the
// cursor for the next page if the extra row proved there is one.
func page[T any](items []T, limit int, keyOf func(T) store.Keyset) ([]T, *string) {
	if len(items) <= limit {
		return items, nil
	}
	items = items[:limit]
	next := encodeCursor(keyOf(items[limit-1]))
	return items, &next
}
