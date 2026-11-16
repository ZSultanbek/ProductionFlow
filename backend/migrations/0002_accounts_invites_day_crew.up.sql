-- 0002: everything Release 1 (US-01..US-05) needs. See docs/architecture.md §3.
--
--   app_user                  accounts (US-01)
--   participant.user_id       links a participant to an account — nullable,
--                             because a participant may never register
--   project.invite_code       one reusable invite code per project (US-02/03)
--   shooting_day_participant  people on set for the whole day (call sheet, US-05)

BEGIN;

CREATE TABLE app_user (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Stored as typed; uniqueness and lookups are case-insensitive via the
    -- index below, so "Aisha@Mail.kz" and "aisha@mail.kz" are one account.
    email         text        NOT NULL CHECK (position('@' in email) > 1),
    password_hash text        NOT NULL,
    display_name  text        NOT NULL CHECK (length(btrim(display_name)) > 0),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX app_user_email_idx ON app_user (lower(email));

-- SET NULL, not CASCADE: deleting an account must not erase the person from the
-- call sheets and impact lists they appear in.
ALTER TABLE participant
    ADD COLUMN user_id uuid REFERENCES app_user (id) ON DELETE SET NULL;

-- One participant row per account per project.
CREATE UNIQUE INDEX participant_project_user_idx
    ON participant (project_id, user_id)
    WHERE user_id IS NOT NULL;

-- 32 hex characters from a random UUID: 122 bits, unguessable. A volatile
-- default is evaluated per row, so existing projects each get their own code.
ALTER TABLE project
    ADD COLUMN invite_code text NOT NULL DEFAULT replace(gen_random_uuid()::text, '-', '');

CREATE UNIQUE INDEX project_invite_code_idx ON project (invite_code);

-- The "assigned directly to the day" half of the impact rule: the gaffer and the
-- sound recordist are on set all day whichever scenes are shot.
CREATE TABLE shooting_day_participant (
    shooting_day_id uuid        NOT NULL REFERENCES shooting_day (id) ON DELETE CASCADE,
    participant_id  uuid        NOT NULL REFERENCES participant (id) ON DELETE CASCADE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (shooting_day_id, participant_id)
);

-- The other direction: "which days is this person on set for" (my schedule).
CREATE INDEX shooting_day_participant_participant_idx
    ON shooting_day_participant (participant_id);

COMMIT;
