---
id: decision-graph-now-enabled
title: "Architecture: NeuroBoost now uses `graph/` for memory management (reverses prior deliberate choice)"
type: decision
status: proposed
proposed_by: consolidator
proposed_at: 2026-08-10
tags: [neuroboost, architecture, memory-management, project-meta]
weight: { importance: 4, connectivity: 3, access: 1, last_accessed: 2026-08-10 }
sources:
  - file: ".remember/night-loop-2026-08-10.md lines 12–36"
  - file: "CLAUDE.md §Memory — **new** (was: 'This project has **no `graph/`**, deliberately')"
stakes: high
links:
  - related-to: memory-split-claude-graph-remember
---

**Summary:** `CLAUDE.md` is being rewritten to reflect that NeuroBoost now maintains a `graph/` directory (same as other ventures: V001, V004). Prior guidance stated deliberately no graph.

⚠ **Правка 2026-08-10:** исходная формулировка узла гласила «this reversal was approved by Denis
2026-08-10 ~03:45». Такого одобрения не было. Дословно он сказал: *«we need to implement graph
knowledge and docs lint or whatever we made for large doc projects so subagents can quickly see
what doc is what for neuroboost»* — это поручение завести граф, но не утверждение формулировки
«отмена прежнего решения». Узел остаётся `status: proposed` до слова Дениса.

**Split of responsibilities:**
- `CLAUDE.md` § Memory: "current-state knowledge lives in `graph/` under the memory-graph skill; this file holds stable rules only"
- `graph/`: current state via nodes (decisions, learnings, entities, work-items) managed by memory-graph skill
- `.remember/`: session journals (`loop-state.md`, `now.md`, etc.) — ephemeral but preserved for reference

**Action items documented in night-loop ~line 25:**
1. §9 (iteration checklist) → replace `/handoff` + `/compact` with memory-graph ritual: INGEST findings as nodes, record recalls in `graph/log.md`, run `graph_weights.py`, LINT
2. §10 (reporting) → separate findings (nodes) from session journals (`.remember/loop-state.md`)
3. §3 (repo facts) → document that graph index injects on session start; answer from it, don't re-read `ROADMAP.md`
4. §2 (locked decisions) → align with any rules PM-Claude changed
