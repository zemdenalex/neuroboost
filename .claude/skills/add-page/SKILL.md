---
name: add-page
description: Add a new frontend page — component, route, navigation link
---

# Add a New Frontend Page

Follow this workflow when adding a new page to NeuroBoost.

## Steps

### 1. Create Page Component

Create `web/src/pages/{Name}/{Name}Page.tsx`:

```typescript
export default function ExamplePage() {
  const { user } = useAuth();

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Page Title</h1>
      {/* Page content */}
    </div>
  );
}
```

- Use `useAuth()` from `contexts/AuthContext` for user data
- Use API client from `api/` for data fetching
- Use Tailwind CSS for styling, Lucide React for icons
- Dark theme is default — use zinc palette colors

### 2. Add Route

In `web/src/router.tsx`, add:

```typescript
{
  path: '/example',
  element: <ProtectedRoute><ExamplePage /></ProtectedRoute>,
}
```

- Use `<ProtectedRoute>` wrapper for authenticated pages
- Use lazy loading for non-critical pages

### 3. Add Navigation Link

In `web/src/components/Layout/NavigationMenu.tsx`, add a menu item:

```typescript
{ path: '/example', label: 'Example', icon: ExampleIcon }
```

### 4. Verify

```bash
cd web && pnpm typecheck && pnpm build
```

Check that the page renders correctly at the route URL.
