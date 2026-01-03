# NeuroBoost

> Calendar-first productivity app for neurodivergent users

**Status:** v0.4.0 - Active Development  
**Live:** [neuroboost.website](https://neuroboost.website)

---

## What is NeuroBoost?

NeuroBoost is a "pushy personal assistant" that helps you stay focused on what matters. Unlike passive to-do lists, it actively nudges you toward your goals through respectful reminders, intelligent task suggestions, and honest reflections.

**For:** Neurodivergent folks, students, freelancers — anyone who knows their goals but struggles in the moment.

**Philosophy:** Calendar-first, task-smart. The calendar is truth; tasks exist to support scheduling and reflection.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.22 + chi router |
| Database | PostgreSQL 16 |
| Frontend | React 18 + TypeScript + Vite + Tailwind |
| Bot | gotgbot (Telegram) |
| Deploy | Docker + Nginx + Let's Encrypt |

---

## Quick Start (Development)

```bash
# Clone
git clone https://github.com/zemdenalex/neuroboost.git
cd neuroboost

# Copy environment
cp .env.example .env
# Edit .env with your values

# Start services
docker compose up -d

# Check health
curl http://localhost:8080/api/health
```

---

## Project Structure

```
neuroboost/
├── api-go/          # Go backend API
├── web/             # React frontend
├── docs/            # Documentation
├── scripts/         # Deployment scripts
└── docker-compose.yml
```

---

## Documentation

- [Development Guide](docs/NEUROBOOST_DEV_GUIDE.md) - How to contribute
- [Implementation Plan](docs/IMPLEMENTATION_PLAN.md) - Current roadmap
- [GitHub Guide](docs/GITHUB_FEATURES_GUIDE.md) - Git workflow
- [Feature List](docs/NeuroBoost_v0_4_0_Feature_List.md) - All planned features

---

## Current Version: v0.4.0.x

### Done
- ✅ Server infrastructure (Docker, Nginx, SSL)
- ✅ PostgreSQL database with migrations
- ✅ Authentication backend (Telegram + Email/Password)
- ✅ JWT middleware
- ✅ Health checks

### In Progress
- 🔧 Login page frontend
- 🔧 Feedback system
- 🔧 Admin panel

### Next
- 📋 Events CRUD
- 📋 Tasks CRUD
- 📋 Calendar UI port from v0.3.x

---

## Contributing

This is currently a solo project, but issues and suggestions are welcome!

1. Check [Feature List](docs/NeuroBoost_v0_4_0_Feature_List.md) for planned work
2. Open an issue to discuss before starting
3. Follow the [Development Guide](docs/NEUROBOOST_DEV_GUIDE.md)

---

## License

Private project. All rights reserved.

---

*Built with ❤️ for the neurodivergent community*