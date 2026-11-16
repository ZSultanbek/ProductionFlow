package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
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

// A cursor is the position of the last item on a page: base64 of a small JSON
// object. Clients must not build or parse it — docs/api.md says so — which
// leaves us free to change what is inside. Each list has its own cursor struct
// matching the columns that list is ordered by.
//
// A cursor arrives from the outside, so every field is checked before it gets
// anywhere near SQL: see the valid() methods.

// projectCursor: projects are ordered by created_at, id (newest first).
type projectCursor struct {
	CreatedAt time.Time `json:"t"`
	ID        string    `json:"id"`
}

func (c projectCursor) valid() bool { return !c.CreatedAt.IsZero() && isUUID(c.ID) }

// dayCursor: days are ordered by shoot_date, call_time, id.
type dayCursor struct {
	ShootDate string `json:"d"`
	CallTime  string `json:"c"`
	ID        string `json:"id"`
}

func (c dayCursor) valid() bool {
	return domain.IsDate(c.ShootDate) && domain.IsCallTime(c.CallTime) && isUUID(c.ID)
}

func encodeCursor(c any) string {
	raw, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// parsePage reads ?cursor= and ?limit=. after is nil for the first page.
func parsePage[C interface{ valid() bool }](r *http.Request) (after *C, limit int, errs domain.FieldErrors) {
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
		var c C
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || json.Unmarshal(decoded, &c) != nil || !c.valid() {
			errs["cursor"] = "is not a cursor this API issued"
		} else {
			after = &c
		}
	}

	return after, limit, errs
}

// page trims a result fetched with limit+1 rows down to limit, and returns the
// cursor for the next page if the extra row proved there is one.
func page[T any](items []T, limit int, cursorOf func(T) any) ([]T, *string) {
	if len(items) <= limit {
		return items, nil
	}
	items = items[:limit]
	next := encodeCursor(cursorOf(items[limit-1]))
	return items, &next
}
