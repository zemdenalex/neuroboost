-- Merge Denis's Telegram identity into his email identity.
--
-- Why now, before the release: production sits on migration 000008, so the
-- `calendar` table does not exist yet. Once 000012 runs it creates one personal
-- calendar PER USER ROW, and merging afterwards means moving calendars, deleting
-- membership rows and stepping around `event.calendar_id`, which is declared
-- without ON DELETE. Doing it here is a straight row move.
--
-- KEEP  : ce841d6a — email identity, 15 events, 3 tasks, created 2026-01-03
-- ABSORB: c514597a — Telegram identity (tg_id 495598685), 7 events, 4 tasks
--
-- 🔴 NOT TOUCHED: 46f66faa "Lizok" is a different person, and the four empty
-- rows (50e80b5d, 356010f6 "Debug", 052f64f2, 9af27563) own nothing. Deleting
-- those is a separate decision and is deliberately not bundled here — an
-- irreversible cleanup riding along with a merge is how the wrong row goes.
--
-- Run inside a transaction and read the verification output BEFORE committing.

BEGIN;

\set keep  'ce841d6a-a5e8-4c4c-8732-48f18b9845dd'
\set absorb 'c514597a-70ad-4d6d-ac9e-934f47b52c89'

-- Every table that references "user"(id), as listed by information_schema on
-- production. Enumerated rather than generated: a loop over the catalogue would
-- silently pick up whatever a future migration adds, and this script is meant to
-- be read and approved once, for exactly these tables.
UPDATE event            SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE event_exception  SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE task             SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE task_dependency  SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE task_requirement SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE reminder         SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE reflection       SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE feedback         SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE need             SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE opportunity      SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE alert_status     SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE pattern_metrics  SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE planning_node    SET user_id = :'keep' WHERE user_id = :'absorb';
UPDATE planning_edge    SET user_id = :'keep' WHERE user_id = :'absorb';

-- The point of the whole exercise: one row that is reachable by both email and
-- Telegram. tg_id is UNIQUE, so the old holder must let go first — hence the
-- delete before the assignment, not after.
DELETE FROM "user" WHERE id = :'absorb';

UPDATE "user" SET tg_id = 495598685 WHERE id = :'keep';

-- Verification. Expect: 22 events, 7 tasks on the kept row; zero rows anywhere
-- still pointing at the absorbed id; exactly one row carrying that tg_id.
SELECT 'events on keep'  AS check, count(*) AS n FROM event    WHERE user_id = :'keep'
UNION ALL SELECT 'tasks on keep',        count(*) FROM task     WHERE user_id = :'keep'
UNION ALL SELECT 'orphans (must be 0)',  count(*) FROM event    WHERE user_id = :'absorb'
UNION ALL SELECT 'users with that tg',   count(*) FROM "user"   WHERE tg_id = 495598685
UNION ALL SELECT 'users total',          count(*) FROM "user";

-- 🔴 Dry run by default. Nothing above is kept unless you ask for it.
--
-- Not left to psql's own end-of-input behaviour: it differs between piped and
-- interactive input, and a script whose safety rests on an implicit default is
-- one that eventually applies when nobody meant it to. The choice is explicit
-- and visible in the output.
--
--   dry run : psql ... -f scripts/merge-identities.sql
--   apply   : psql ... -v apply=1 -f scripts/merge-identities.sql
\if :{?apply}
  COMMIT;
  \echo '>>> COMMITTED — the merge is permanent.'
\else
  ROLLBACK;
  \echo '>>> ROLLED BACK — dry run. Read the numbers above; re-run with -v apply=1 to keep them.'
\endif
