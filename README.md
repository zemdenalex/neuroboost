# NeuroBoost v0.4.0 — Skeleton

This repository contains a **complete project skeleton** for NeuroBoost v0.4.0.
Everything returns **stubs** (501 Not Implemented) except the health check.

## What’s here
- **Backend (Go + chi)** in `neuroboost-api-go/` with **36 endpoints registered**.
- **Frontend (React + TS + Vite + Tailwind)** in `neuroboost-web/` with pages as **thin orchestration** and components in **implementation layer**.
- **Database** migrations (18 files) mirroring the schema.
- **Infrastructure**: Docker Compose + nginx.
- **Docs**: `PROGRESS.md` checklist.

## Quick start
```bash
# (Optionally) build artifacts locally
docker-compose up -d
curl -s http://localhost:8080/api/health

# Frontend (dev): open http://localhost:5173
# Backend: http://localhost:8080/api/*
```

## Conventions
- Pages orchestrate; Components implement.
- Handlers return **501** JSON stubs via a shared helper.
- JWT middleware is a stub that **always passes** (dev only).
