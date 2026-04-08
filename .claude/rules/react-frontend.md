---
description: React frontend conventions for NeuroBoost web app
paths:
  - "web/**"
---

# React Frontend Rules

- Use the centralized API client: `import { api } from '../api/client'` — never raw `fetch`
- snake_case from API responses, camelCase in frontend code — conversion happens in API client layer
- Use Lucide React for icons — never Material UI or other icon libraries
- Use Tailwind CSS for styling — dark theme (zinc palette) is default
- Use `useAuth()` from `contexts/AuthContext.tsx` for authentication state
- Complex components: split into subfiles like `components/Calendar/WeekGrid/` (types, utils, constants, hooks, subcomponents)
- Simple components: single file with props interface
- No `any` in TypeScript — use proper types from `src/types/`
- Modal pattern: Escape key + click-outside to close
- Run `pnpm typecheck` after any changes
- Run `pnpm build` before marking work done
