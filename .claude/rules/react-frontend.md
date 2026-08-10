---
description: React frontend conventions for NeuroBoost web app
paths:
  - "web/**"
---

# React Frontend Rules

- Use the centralized API client: `import { api } from '../api/client'` — never raw `fetch`
- 🔴 snake_case from the API is **not** converted automatically — conversion is per-endpoint and
  easy to forget. That omission WAS bug T1: `getTasks` cast snake_case to a camelCase type, so
  `estimatedMinutes` was always `undefined` and every dragged task became 60 minutes. Converters
  live next to their API module (`web/src/api/toTask.ts`, `types/index.ts` → `toNbEvent`)
- ⚠ Two parallel event API stacks: `api/events.ts` (snake_case, used by Pomodoro) and
  `api/index.ts` (camelCase, used by the calendar). Both export `moveEvent` hitting DIFFERENT
  endpoints. Check which one you are importing
- Use Lucide React for icons — never Material UI or other icon libraries
- Use Tailwind CSS for styling — dark theme (zinc palette) is default
- Use `useAuthContext()` from `contexts/AuthContext.tsx` for authentication state (there is no
  `useAuth()`)
- Complex components: split into subfiles like `components/Calendar/WeekGrid/` (types, utils, constants, hooks, subcomponents)
- Simple components: single file with props interface
- No `any` in TypeScript — use proper types from `src/types/`
- Modal pattern: Escape key + click-outside to close
- Run `pnpm typecheck` after any changes
- Run `pnpm build` before marking work done
