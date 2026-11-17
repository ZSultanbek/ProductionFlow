package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// SceneFilter narrows a project's scene list. The zero value is every scene.
type SceneFilter struct {
	ShootingDayID   string // only the scenes on this day
	UnscheduledOnly bool   // only the scenes on no day
}

// sceneColumns: coalesce turns "no day" into "", the way domain.Scene spells
// it. Callers alias scene as s.
const sceneColumns = `s.id::text, s.project_id::text, coalesce(s.shooting_day_id::text, ''),
	s.number, s.description`

func scanScene(row pgx.Row) (domain.Scene, error) {
	var sc domain.Scene
	err := row.Scan(&sc.ID, &sc.ProjectID, &sc.ShootingDayID, &sc.Number, &sc.Description)
	return sc, err
}

// querier is what both the pool and a transaction can do, so a read can run
// inside a write's transaction or on its own.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ScenesForProject lists a project's scenes by number, each with its people.
// Ordered by number as text ("12A" before "2"), as docs/api.md says; the
// client sorts naturally for display.
func (db *DB) ScenesForProject(ctx context.Context, projectID string, f SceneFilter) ([]domain.Scene, error) {
	var dayID *string
	if f.ShootingDayID != "" {
		dayID = &f.ShootingDayID
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT `+sceneColumns+`
		FROM scene s
		WHERE s.project_id = $1
		  AND ($2::uuid IS NULL OR s.shooting_day_id = $2::uuid)
		  AND (NOT $3 OR s.shooting_day_id IS NULL)
		ORDER BY `+sceneOrder,
		projectID, dayID, f.UnscheduledOnly)
	if err != nil {
		return nil, fmt.Errorf("scenes for project: %w", err)
	}
	scenes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Scene, error) {
		return scanScene(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scenes for project: %w", err)
	}
	if err := attachSceneParticipants(ctx, db.Pool, scenes); err != nil {
		return nil, fmt.Errorf("scenes for project: %w", err)
	}
	return scenes, nil
}

// CreateScene adds a scene, on a day or unscheduled. The caller has checked
// that the day, if any, belongs to the same project.
func (db *DB) CreateScene(ctx context.Context, projectID string, in domain.SceneInput) (domain.Scene, error) {
	var sc domain.Scene
	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		// A new scene on a day changes who that day affects.
		if err := lockDays(ctx, tx, in.ShootingDayID); err != nil {
			return err
		}
		var err error
		sc, err = scanScene(tx.QueryRow(ctx, `
			INSERT INTO scene AS s (project_id, shooting_day_id, number, description)
			VALUES ($1, nullif($2, '')::uuid, $3, $4)
			RETURNING `+sceneColumns,
			projectID, in.ShootingDayID, in.Number, in.Description))
		return err
	})
	if isUniqueViolation(err, "scene_project_id_number_key") {
		return domain.Scene{}, ErrSceneNumberTaken
	}
	if err != nil {
		return domain.Scene{}, fmt.Errorf("create scene: %w", err)
	}
	sc.Participants = []domain.ParticipantRef{}
	return sc, nil
}

// SceneForUser returns a scene with its people, and the user's membership of
// its project. ErrNotFound if there is no such scene or the user is not in its
// project.
func (db *DB) SceneForUser(ctx context.Context, userID, sceneID string) (domain.Scene, domain.Membership, error) {
	var (
		sc   domain.Scene
		m    domain.Membership
		role string
	)
	err := db.Pool.QueryRow(ctx, `
		SELECT `+sceneColumns+`, `+membershipColumns+`
		FROM scene s
		JOIN project p ON p.id = s.project_id
		JOIN participant pa ON pa.project_id = s.project_id AND pa.user_id = $1
		WHERE s.id = $2`,
		userID, sceneID,
	).Scan(&sc.ID, &sc.ProjectID, &sc.ShootingDayID, &sc.Number, &sc.Description,
		&m.Project.ID, &m.Project.Name, &m.Project.InviteCode, &m.Project.CreatedAt, &m.ParticipantID, &role)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidText(err) {
		return domain.Scene{}, domain.Membership{}, ErrNotFound
	}
	if err != nil {
		return domain.Scene{}, domain.Membership{}, fmt.Errorf("scene for user: %w", err)
	}
	m.Role = domain.Role(role)

	scenes := []domain.Scene{sc}
	if err := attachSceneParticipants(ctx, db.Pool, scenes); err != nil {
		return domain.Scene{}, domain.Membership{}, fmt.Errorf("scene for user: %w", err)
	}
	return scenes[0], m, nil
}

// UpdateScene applies a patch and records what changed as one change event,
// in the same transaction. A patch that changes nothing writes nothing — not
// even an event (docs/api.md). The caller has checked that a new day belongs
// to the scene's project.
func (db *DB) UpdateScene(ctx context.Context, actorParticipantID, sceneID string, p domain.ScenePatch) (domain.Scene, error) {
	var sc domain.Scene
	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		current, err := scanScene(tx.QueryRow(ctx, `
			SELECT `+sceneColumns+` FROM scene s WHERE s.id = $1 FOR UPDATE`, sceneID))
		if errors.Is(err, pgx.ErrNoRows) || isInvalidText(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		diff := domain.DiffScene(current, p)
		if len(diff) == 0 {
			sc = current
			return nil
		}

		// Moving a scene changes who both its old and its new day affect.
		newDay := current.ShootingDayID
		if p.SetDay {
			newDay = p.ShootingDayID
		}
		if err := lockDays(ctx, tx, current.ShootingDayID, newDay); err != nil {
			return err
		}

		sc, err = scanScene(tx.QueryRow(ctx, `
			UPDATE scene AS s SET
				shooting_day_id = CASE WHEN $2 THEN nullif($3, '')::uuid ELSE s.shooting_day_id END,
				description     = CASE WHEN $4 THEN $5 ELSE s.description END,
				updated_at      = now()
			WHERE s.id = $1
			RETURNING `+sceneColumns,
			sceneID, p.SetDay, p.ShootingDayID, p.SetDescription, p.Description))
		if err != nil {
			return err
		}

		raw, err := json.Marshal(diff)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO change_event (project_id, actor_participant_id, entity_type, entity_id, diff)
			VALUES ($1, $2, 'scene', $3, $4)`,
			sc.ProjectID, actorParticipantID, sc.ID, raw)
		return err
	})
	if errors.Is(err, ErrNotFound) {
		return domain.Scene{}, ErrNotFound
	}
	if err != nil {
		return domain.Scene{}, fmt.Errorf("update scene: %w", err)
	}

	scenes := []domain.Scene{sc}
	if err := attachSceneParticipants(ctx, db.Pool, scenes); err != nil {
		return domain.Scene{}, fmt.Errorf("update scene: %w", err)
	}
	return scenes[0], nil
}

// SetSceneParticipants replaces who is needed in a scene. The caller has
// checked that every id is a participant of the scene's project.
func (db *DB) SetSceneParticipants(ctx context.Context, sceneID string, participantIDs []string) (domain.Scene, error) {
	var sc domain.Scene
	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		sc, err = scanScene(tx.QueryRow(ctx, `
			SELECT `+sceneColumns+` FROM scene s WHERE s.id = $1 FOR UPDATE`, sceneID))
		if errors.Is(err, pgx.ErrNoRows) || isInvalidText(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := lockDays(ctx, tx, sc.ShootingDayID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `DELETE FROM scene_participant WHERE scene_id = $1`, sceneID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO scene_participant (scene_id, participant_id)
			SELECT $1, unnest($2::uuid[])`,
			sceneID, participantIDs); err != nil {
			return err
		}

		scenes := []domain.Scene{sc}
		if err := attachSceneParticipants(ctx, tx, scenes); err != nil {
			return err
		}
		sc = scenes[0]
		return nil
	})
	if errors.Is(err, ErrNotFound) {
		return domain.Scene{}, ErrNotFound
	}
	if err != nil {
		return domain.Scene{}, fmt.Errorf("set scene participants: %w", err)
	}
	return sc, nil
}

// ImpactInput loads everything domain.ComputeImpact looks at for one day: its
// scenes, who is in them, who is on set all day, and the project's people.
// It only loads; the rule itself is in domain, where it is unit-tested.
func (db *DB) ImpactInput(ctx context.Context, day domain.ShootingDay) (domain.ImpactInput, error) {
	in, err := loadImpactInput(ctx, db.Pool, day)
	if err != nil {
		return domain.ImpactInput{}, fmt.Errorf("impact input: %w", err)
	}
	return in, nil
}

func loadImpactInput(ctx context.Context, q querier, day domain.ShootingDay) (domain.ImpactInput, error) {
	var in domain.ImpactInput

	rows, err := q.Query(ctx, `
		SELECT `+sceneColumns+` FROM scene s WHERE s.shooting_day_id = $1`, day.ID)
	if err != nil {
		return in, err
	}
	in.Scenes, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Scene, error) {
		return scanScene(row)
	})
	if err != nil {
		return in, err
	}

	rows, err = q.Query(ctx, `
		SELECT sp.scene_id::text, sp.participant_id::text
		FROM scene_participant sp
		JOIN scene s ON s.id = sp.scene_id
		WHERE s.shooting_day_id = $1`, day.ID)
	if err != nil {
		return in, err
	}
	in.SceneLinks, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.SceneLink, error) {
		var l domain.SceneLink
		err := row.Scan(&l.SceneID, &l.ParticipantID)
		return l, err
	})
	if err != nil {
		return in, err
	}

	rows, err = q.Query(ctx, `
		SELECT participant_id::text FROM shooting_day_participant WHERE shooting_day_id = $1`, day.ID)
	if err != nil {
		return in, err
	}
	in.DayLinks, err = pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return in, err
	}

	rows, err = q.Query(ctx, `
		SELECT `+participantColumns+` FROM participant pa WHERE pa.project_id = $1`, day.ProjectID)
	if err != nil {
		return in, err
	}
	in.Participants, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Participant, error) {
		return scanParticipant(row)
	})
	return in, err
}

// attachSceneParticipants fills in Participants for each scene with one query,
// in call-sheet order. Every scene gets a non-nil list, so an empty one is []
// in JSON.
func attachSceneParticipants(ctx context.Context, q querier, scenes []domain.Scene) error {
	if len(scenes) == 0 {
		return nil
	}
	ids := make([]string, len(scenes))
	index := make(map[string]int, len(scenes))
	for i := range scenes {
		ids[i] = scenes[i].ID
		index[scenes[i].ID] = i
		scenes[i].Participants = []domain.ParticipantRef{}
	}

	rows, err := q.Query(ctx, `
		SELECT sp.scene_id::text, pa.id::text, pa.display_name, pa.role
		FROM scene_participant sp
		JOIN participant pa ON pa.id = sp.participant_id
		WHERE sp.scene_id = ANY($1::uuid[])
		ORDER BY `+peopleOrder,
		ids)
	if err != nil {
		return fmt.Errorf("scene participants: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			sceneID string
			ref     domain.ParticipantRef
			role    string
		)
		if err := rows.Scan(&sceneID, &ref.ID, &ref.DisplayName, &role); err != nil {
			return fmt.Errorf("scene participants: %w", err)
		}
		ref.Role = domain.Role(role)
		i := index[sceneID]
		scenes[i].Participants = append(scenes[i].Participants, ref)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scene participants: %w", err)
	}
	return nil
}
