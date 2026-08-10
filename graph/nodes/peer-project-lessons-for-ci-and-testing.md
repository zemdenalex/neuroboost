---
id: peer-project-lessons-for-ci-and-testing
title: "Five peer-project rules applied to NeuroBoost night loops and verification"
type: learning
status: proposed
proposed_by: consolidator
proposed_at: 2026-08-10
tags: [neuroboost, testing, ci-cd, verification, lessons-learned]
weight: { importance: 3, connectivity: 2, access: 1, last_accessed: 2026-08-10 }
sources:
  - file: ".remember/night-loop-2026-08-10.md §8-бис (rules from FilinTermy, Archifex, AVOCAR, Nivium, KaUnion)"
stakes: medium
links:
  - related-to: workitem-p2-notifications-last-mile
---

**Summary:** Five explicit rules extracted from neighbouring projects and documented for NeuroBoost's night-loop work.

**1. FilinTermy — verify cycles cost real time; build a local harness first**
- Cost of deploy-cycle verification: ~4 hours + ~1M tokens per section
- Rule: **Check locally first** (docker compose + real creds) before shipping to server
- Application: for Telegram notifications, cycle is "fix → push → GitHub Actions → server → check logs → repeat". Instant-fail if SERVICE_TOKEN is wrong. Build local proof first, then deploy known-good to prod.

**2. Archifex — doc drift kills truth**
- Symptom: ROADMAP says "Phase 3 not started" months after 3a/3b merged
- Rule: **Count the filesystem, never trust a doc line** — verify state via `git log`, file contents, live queries, not docs
- Application: ROADMAP.md dates can rot. Rely on `docs/ROADMAP.md` only for roadmap structure; confirm release tag, branch state, migration count via git/filesystem.

**3. AVOCAR — structural checks don't replace live testing**
- Symptom: Playwright MCP kept failing; live checks got swapped for "passes JSON structure"
- Rule: **Controls that cannot fail are not confirmations.** "File parsed OK" ≠ "feature works"
- Application for R1 dialog: checked JSON keys exist (✓), but dialog rendering in browser **never tested** (browser test harness doesn't exist). For Telegram, live message in phone > "log is clean" > "endpoint responds".

**4. AVOCAR + Nivium — key hygiene**
- Symptom: root-key leaked in transcript; flagged for rotation
- Rule: **Do not print secrets to reports, terminal, artifact, or git.** Read them in server commands; store new tokens in memory-dir only, outside git.
- Application: SERVICE_TOKEN generated once, stored in `~/.claude/projects/.../memory/`, passed to server via SSH, never echoed in logs.

**5. KaUnion — verify by executing, not reading**
- Symptom: scoring algorithm read-verified; execution found errors logic didn't catch
- Rule: **Verify by execution, not by re-reading code.** For reminders, create event + wait for message (execute) not "reread DueReminders logic" (audit).
- Application: proof-of-Telegram is "message landed on phone with timestamp + message_id", not "code review says pull → send → ack looks right".
