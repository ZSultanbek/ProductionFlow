package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// DayKeyset is a position in a list of days ordered by date, call time, id.
type DayKeyset struct {
	ShootDate string // YYYY-MM-DD
	CallTime  string // HH:MM
	ID        string
}

// dayColumns formats the date and call time in SQL, so they reach Go as the
// exact strings the API sends ("2026-10-04", "07:30") and never pass through a
// time.Time that could attach a time zone to them. Callers alias shooting_day
// as d.
const dayColumns = `d.id::text, d.project_id::text,
	to_char(d.shoot_date, 'YYYY-MM-DD'), to_char(d.call_time, 'HH24:MI'),
	d.location, d.updated_at`

// roleOrder sorts people the way a call sheet lists them: producers, then crew,
// then cast. Callers alias participant as pa.
const roleOrder = `array_position(ARRAY['producer', 'crew', 'cast'], pa.role)`

// CreateDay adds a shooting day to a project. The caller has already checked
// that the user is a producer there and that the input is valid.
func (db *DB) CreateDay(ctx context.Context, projectID string, in domain.DayInput) (domain.ShootingDay, error) {
	var d domain.ShootingDay
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO shooting_day AS d (project_id, shoot_date, call_time, location)
		VALUES ($1, $2::date, $3::time, $4)
		RETURNING `+dayColumns,
		projectID, in.ShootDate, in.CallTime, in.Location,
	).Scan(&d.ID, &d.ProjectID, &d.ShootDate, &d.CallTime, &d.Location, &d.UpdatedAt)
	if err != nil {
		return domain.ShootingDay{}, fmt.Errorf("create day: %w", err)
	}
	return d, nil
}

// DaysForProject lists a project's days in shooting order. from, if not empty,
// skips days before that date. Pass after = nil for the first page.
func (db *DB) DaysForProject(ctx context.Context, projectID, from string, after *DayKeyset, limit int) ([]domain.ShootingDay, error) {
	var fromDate, afterDate, afterTime, afterID *string
	if from != "" {
		fromDate = &from
	}
	if after != nil {
		afterDate, afterTime, afterID = &after.ShootDate, &after.CallTime, &after.ID
	}

	// Two units can shoot on the same date at the same call time, so the id is
	// part of the ordering: without it, paging could skip or repeat a day.
	rows, err := db.Pool.Query(ctx, `
		SELECT `+dayColumns+`,
		       (SELECT count(*) FROM scene s WHERE s.shooting_day_id = d.id)
		FROM shooting_day d
		WHERE d.project_id = $1
		  AND ($2::date IS NULL OR d.shoot_date >= $2::date)
		  AND ($3::date IS NULL OR (d.shoot_date, d.call_time, d.id) > ($3::date, $4::time, $5::uuid))
		ORDER BY d.shoot_date, d.call_time, d.id
		LIMIT $6`,
		projectID, fromDate, afterDate, afterTime, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("days for project: %w", err)
	}

	days, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.ShootingDay, error) {
		var d domain.ShootingDay
		err := row.Scan(&d.ID, &d.ProjectID, &d.ShootDate, &d.CallTime, &d.Location, &d.UpdatedAt, &d.SceneCount)
		return d, err
	})
	if err != nil {
		return nil, fmt.Errorf("days for project: %w", err)
	}
	return days, nil
}

// CallSheetForUser returns one day with its scenes and people. ErrNotFound if
// the day does not exist, the id is not a UUID, or the user is not in the day's
// project — the same answer for all three.
func (db *DB) CallSheetForUser(ctx context.Context, userID, dayID string) (domain.CallSheet, error) {
	var (
		sheet domain.CallSheet
		role  string
	)

	// The join on participant is the access check: no membership, no row.
	err := db.Pool.QueryRow(ctx, `
		SELECT `+dayColumns+`, p.name, pa.role
		FROM shooting_day d
		JOIN project p ON p.id = d.project_id
		JOIN participant pa ON pa.project_id = d.project_id AND pa.user_id = $1
		WHERE d.id = $2`,
		userID, dayID,
	).Scan(&sheet.Day.ID, &sheet.Day.ProjectID, &sheet.Day.ShootDate, &sheet.Day.CallTime,
		&sheet.Day.Location, &sheet.Day.UpdatedAt, &sheet.ProjectName, &role)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidText(err) {
		return domain.CallSheet{}, ErrNotFound
	}
	if err != nil {
		return domain.CallSheet{}, fmt.Errorf("call sheet: load day: %w", err)
	}
	sheet.MyRole = domain.Role(role)

	// People on set for the whole day, whatever is being shot.
	rows, err := db.Pool.Query(ctx, `
		SELECT pa.id::text, pa.display_name, pa.role
		FROM shooting_day_participant sdp
		JOIN participant pa ON pa.id = sdp.participant_id
		WHERE sdp.shooting_day_id = $1
		ORDER BY `+roleOrder+`, pa.display_name, pa.id`,
		dayID)
	if err != nil {
		return domain.CallSheet{}, fmt.Errorf("call sheet: on set all day: %w", err)
	}
	sheet.OnSetAllDay, err = pgx.CollectRows(rows, scanParticipantRef)
	if err != nil {
		return domain.CallSheet{}, fmt.Errorf("call sheet: on set all day: %w", err)
	}

	// The day's scenes, then everyone in them in one query, grouped in Go.
	rows, err = db.Pool.Query(ctx, `
		SELECT id::text, number, description
		FROM scene
		WHERE shooting_day_id = $1
		ORDER BY number, id`,
		dayID)
	if err != nil {
		return domain.CallSheet{}, fmt.Errorf("call sheet: scenes: %w", err)
	}
	sheet.Scenes, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.CallSheetScene, error) {
		var s domain.CallSheetScene
		err := row.Scan(&s.ID, &s.Number, &s.Description)
		return s, err
	})
	if err != nil {
		return domain.CallSheet{}, fmt.Errorf("call sheet: scenes: %w", err)
	}

	sceneIndex := make(map[string]int, len(sheet.Scenes))
	for i, s := range sheet.Scenes {
		sceneIndex[s.ID] = i
	}

	rows, err = db.Pool.Query(ctx, `
		SELECT sp.scene_id::text, pa.id::text, pa.display_name, pa.role
		FROM scene s
		JOIN scene_participant sp ON sp.scene_id = s.id
		JOIN participant pa ON pa.id = sp.participant_id
		WHERE s.shooting_day_id = $1
		ORDER BY `+roleOrder+`, pa.display_name, pa.id`,
		dayID)
	if err != nil {
		return domain.CallSheet{}, fmt.Errorf("call sheet: scene participants: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			sceneID string
			ref     domain.ParticipantRef
			r       string
		)
		if err := rows.Scan(&sceneID, &ref.ID, &ref.DisplayName, &r); err != nil {
			return domain.CallSheet{}, fmt.Errorf("call sheet: scene participants: %w", err)
		}
		ref.Role = domain.Role(r)
		// A scene added between the two queries is simply not on this sheet.
		if i, ok := sceneIndex[sceneID]; ok {
			sheet.Scenes[i].Participants = append(sheet.Scenes[i].Participants, ref)
		}
	}
	if err := rows.Err(); err != nil {
		return domain.CallSheet{}, fmt.Errorf("call sheet: scene participants: %w", err)
	}

	sheet.Day.SceneCount = len(sheet.Scenes)
	return sheet, nil
}

func scanParticipantRef(row pgx.CollectableRow) (domain.ParticipantRef, error) {
	var (
		ref  domain.ParticipantRef
		role string
	)
	err := row.Scan(&ref.ID, &ref.DisplayName, &role)
	ref.Role = domain.Role(role)
	return ref, err
}

// TodayForUser answers "where do I need to be": every shooting day on date in
// any of the user's projects. If there are none on date, it returns the days on
// the next date that has any. The returned date is "" when nothing is scheduled
// on or after date at all.
//
// date comes from the caller's phone. The server never works out "today"
// itself — it does not know what time zone the user is standing in.
func (db *DB) TodayForUser(ctx context.Context, userID, date string) (string, []domain.AgendaDay, error) {
	var target *string
	err := db.Pool.QueryRow(ctx, `
		SELECT to_char(min(d.shoot_date), 'YYYY-MM-DD')
		FROM shooting_day d
		JOIN participant pa ON pa.project_id = d.project_id
		WHERE pa.user_id = $1 AND d.shoot_date >= $2::date`,
		userID, date).Scan(&target)
	if err != nil {
		return "", nil, fmt.Errorf("today: find date: %w", err)
	}
	if target == nil {
		return "", nil, nil
	}

	// Every day of the user's projects on that date, including ones they have
	// no scene in: a crew member still wants to know the project is shooting.
	rows, err := db.Pool.Query(ctx, `
		SELECT `+dayColumns+`, p.name,
		       EXISTS (SELECT 1 FROM shooting_day_participant sdp
		               WHERE sdp.shooting_day_id = d.id AND sdp.participant_id = pa.id)
		FROM shooting_day d
		JOIN project p ON p.id = d.project_id
		JOIN participant pa ON pa.project_id = d.project_id AND pa.user_id = $1
		WHERE d.shoot_date = $2::date
		ORDER BY d.call_time, p.name, d.id`,
		userID, *target)
	if err != nil {
		return "", nil, fmt.Errorf("today: days: %w", err)
	}
	days, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.AgendaDay, error) {
		var a domain.AgendaDay
		err := row.Scan(&a.Day.ID, &a.Day.ProjectID, &a.Day.ShootDate, &a.Day.CallTime,
			&a.Day.Location, &a.Day.UpdatedAt, &a.ProjectName, &a.OnSetAllDay)
		return a, err
	})
	if err != nil {
		return "", nil, fmt.Errorf("today: days: %w", err)
	}

	dayIndex := make(map[string]int, len(days))
	for i, a := range days {
		dayIndex[a.Day.ID] = i
	}

	// Only the scenes this user is personally needed in.
	rows, err = db.Pool.Query(ctx, `
		SELECT s.shooting_day_id::text, s.id::text, s.number, s.description
		FROM scene s
		JOIN shooting_day d ON d.id = s.shooting_day_id
		JOIN scene_participant sp ON sp.scene_id = s.id
		JOIN participant pa ON pa.id = sp.participant_id
		WHERE pa.user_id = $1 AND d.shoot_date = $2::date
		ORDER BY s.number, s.id`,
		userID, *target)
	if err != nil {
		return "", nil, fmt.Errorf("today: my scenes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			dayID string
			ref   domain.SceneRef
		)
		if err := rows.Scan(&dayID, &ref.ID, &ref.Number, &ref.Description); err != nil {
			return "", nil, fmt.Errorf("today: my scenes: %w", err)
		}
		if i, ok := dayIndex[dayID]; ok {
			days[i].MyScenes = append(days[i].MyScenes, ref)
		}
	}
	if err := rows.Err(); err != nil {
		return "", nil, fmt.Errorf("today: my scenes: %w", err)
	}

	return *target, days, nil
}
