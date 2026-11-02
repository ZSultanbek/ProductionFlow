BEGIN;

DROP TABLE IF EXISTS shooting_day_participant;

DROP INDEX IF EXISTS project_invite_code_idx;
ALTER TABLE project DROP COLUMN IF EXISTS invite_code;

DROP INDEX IF EXISTS participant_project_user_idx;
ALTER TABLE participant DROP COLUMN IF EXISTS user_id;

DROP TABLE IF EXISTS app_user;

COMMIT;
