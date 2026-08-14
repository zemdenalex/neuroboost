-- Restore the three default presets on the staging account.
-- They were flattened to identical values by the preset chooser that used to
-- sit inside the presets editor. Only the `presets` key is touched; every other
-- reminder setting (digest, quiet hours, defaults) is left alone.
BEGIN;

SELECT left(id::text,8) AS id8, settings->'reminders'->'presets' AS before
FROM "user" WHERE settings->'reminders'->'presets' IS NOT NULL;

UPDATE "user"
SET settings = jsonb_set(
      settings,
      '{reminders,presets}',
      '{"важное": [43200, 10080, 4320, 1440, 60], "обычное": [1440, 60], "без": []}'::jsonb,
      true)
WHERE settings->'reminders'->'presets' IS NOT NULL;

SELECT left(id::text,8) AS id8,
       settings->'reminders'->'presets' AS after,
       settings->'reminders'->'digest_at' AS digest_kept,
       settings->'reminders'->'default_event_preset' AS default_kept
FROM "user" WHERE settings->'reminders'->'presets' IS NOT NULL;

\if :{?apply}
  COMMIT;
  \echo '>>> COMMITTED'
\else
  ROLLBACK;
  \echo '>>> ROLLED BACK (dry run) — re-run with -v apply=1'
\endif
