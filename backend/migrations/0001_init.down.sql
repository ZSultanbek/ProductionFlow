-- Reverse of 0001_init. Dropped child-first so the foreign keys allow it.

BEGIN;

DROP TABLE IF EXISTS change_event;
DROP TABLE IF EXISTS scene_participant;
DROP TABLE IF EXISTS participant;
DROP TABLE IF EXISTS scene;
DROP TABLE IF EXISTS shooting_day;
DROP TABLE IF EXISTS project;

COMMIT;
