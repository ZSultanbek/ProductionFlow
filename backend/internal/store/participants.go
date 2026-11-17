package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// ErrParticipantEmailTaken: the email already belongs to another participant of
// the project who is linked to a different account.
var ErrParticipantEmailTaken = errors.New("store: participant email already in project")

// participantColumns: coalesce turns SQL NULL into "" so Go never has to deal
// with *string for "no account" and "no email".
const participantColumns = `id::text, project_id::text, coalesce(user_id::text, ''),
	display_name, coalesce(email, ''), role`

func scanParticipant(row pgx.Row) (domain.Participant, error) {
	var p domain.Participant
	var role string
	err := row.Scan(&p.ID, &p.ProjectID, &p.UserID, &p.DisplayName, &p.Email, &role)
	p.Role = domain.Role(role)
	return p, err
}

// JoinProject adds user to the project as role, or does what domain.DecideJoin
// says instead: return their existing row, or link a row the producer created
// for them by email. created reports whether a new participant was inserted
// (the handler answers 201 for that and 200 otherwise).
//
// The project row is locked first, so two people following the same invite at
// the same moment are handled one after the other rather than racing the
// unique indexes.
func (db *DB) JoinProject(ctx context.Context, user domain.User, projectID string, role domain.Role) (domain.Participant, bool, error) {
	var (
		result  domain.Participant
		created bool
	)

	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		var locked string
		err := tx.QueryRow(ctx, `SELECT id::text FROM project WHERE id = $1 FOR UPDATE`, projectID).Scan(&locked)
		if errors.Is(err, pgx.ErrNoRows) || isInvalidText(err) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		rows, err := tx.Query(ctx, `SELECT `+participantColumns+` FROM participant WHERE project_id = $1`, projectID)
		if err != nil {
			return fmt.Errorf("load participants: %w", err)
		}
		people, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Participant, error) {
			return scanParticipant(row)
		})
		if err != nil {
			return fmt.Errorf("load participants: %w", err)
		}

		outcome, participantID := domain.DecideJoin(people, user.ID, user.Email)

		switch outcome {
		case domain.JoinAlreadyMember:
			for _, p := range people {
				if p.ID == participantID {
					result = p
				}
			}
			return nil

		case domain.JoinLinkExisting:
			// The row keeps the name and role the producer gave it; only the
			// account link is new.
			result, err = scanParticipant(tx.QueryRow(ctx, `
				UPDATE participant
				SET user_id = $1, updated_at = now()
				WHERE id = $2
				RETURNING `+participantColumns,
				user.ID, participantID))
			if err != nil {
				return fmt.Errorf("link participant: %w", err)
			}
			return nil

		case domain.JoinCreate:
			result, err = scanParticipant(tx.QueryRow(ctx, `
				INSERT INTO participant (project_id, user_id, display_name, email, role)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING `+participantColumns,
				projectID, user.ID, user.DisplayName, user.Email, string(role)))
			if err != nil {
				return fmt.Errorf("insert participant: %w", err)
			}
			created = true
			return nil

		default: // domain.JoinEmailConflict
			return ErrParticipantEmailTaken
		}
	})
	if err != nil {
		return domain.Participant{}, false, err
	}
	return result, created, nil
}

// ParticipantsForProject lists everyone in a project the way a call sheet does:
// producers, then crew, then cast, each by name. A crew is 5–15 people, so the
// list is never paged.
func (db *DB) ParticipantsForProject(ctx context.Context, projectID string) ([]domain.Participant, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT `+participantColumns+`
		FROM participant pa
		WHERE pa.project_id = $1
		ORDER BY `+peopleOrder,
		projectID)
	if err != nil {
		return nil, fmt.Errorf("participants for project: %w", err)
	}
	people, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Participant, error) {
		return scanParticipant(row)
	})
	if err != nil {
		return nil, fmt.Errorf("participants for project: %w", err)
	}
	return people, nil
}

// CreateParticipant adds a person by name, with no account (US-06). If they
// register later with the same email and follow the invite, JoinProject links
// them to this row instead of creating a second one.
func (db *DB) CreateParticipant(ctx context.Context, projectID string, in domain.ParticipantInput) (domain.Participant, error) {
	p, err := scanParticipant(db.Pool.QueryRow(ctx, `
		INSERT INTO participant (project_id, display_name, email, role)
		VALUES ($1, $2, nullif($3, ''), $4)
		RETURNING `+participantColumns,
		projectID, in.DisplayName, in.Email, string(in.Role)))
	if isUniqueViolation(err, "participant_project_email_idx") {
		return domain.Participant{}, ErrParticipantEmailTaken
	}
	if err != nil {
		return domain.Participant{}, fmt.Errorf("create participant: %w", err)
	}
	return p, nil
}
