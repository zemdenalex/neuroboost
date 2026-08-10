---
id: memory-split-claude-graph-remember
title: "Rule: three-layer memory split — CLAUDE.md (stable), graph/ (current state), .remember/ (journals)"
type: rule
status: proposed
proposed_by: consolidator
proposed_at: 2026-08-10
tags: [neuroboost, memory-management, project-structure]
weight: { importance: 4, connectivity: 2, access: 2, last_accessed: 2026-08-10 }
sources:
  - file: ".remember/night-loop-2026-08-10.md lines 25–36"
stakes: high
links:
  - related-to: decision-graph-now-enabled
---
**Summary:** NeuroBoost enforces a three-layer split to prevent drift and duplicate-source-of-truth disease (observed in Archifex per §8-бис).

**Layer 1: `CLAUDE.md` — stable rules only**
- Tech stack, naming, commands, conventions
- DO NOT include current status here ("releases X, state is Y, current sprint is Z")
- When you learn something durable (a decision, a preference, a gotcha), check: is this a stable rule? If yes and it's a correction of existing guidance, update CLAUDE.md AND run git commit
- "Counts drift. Count the filesystem, never trust a doc line — including this one." (per existing text)

**Layer 2: `graph/nodes/*.md` — current state via memory-graph skill**
- Decisions made, learnings discovered, work items tracked
- Status: proposed → verified → archived
- Injected into every session start (graph index, not full bodies)
- Survivorship: compaction happens on handoff, not mid-session

**Layer 3: `.remember/` — session journals**
- `now.md` (current session buffer)
- `today-*.md` (daily summaries)
- `recent.md` (7 days)
- `loop-state.md` (for autonomous loops; iteration progress)
- NOT the source of current truth; reference only
- Read when backfilling context, not to understand project state

**Why three layers:**
"Archifex — deiсь по сих пор пишут, что Phase 3 не начата, хотя 3a и 3b смержены в master ещё 5–7 июля." — ROADMAP is a fossil; ANY doc written once stops being maintenance. A single source of truth + graph node updates keep state true.
