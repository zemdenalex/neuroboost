import { RouterProvider } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { ErrorBoundary } from './components/ErrorBoundary'
import { router } from './router'
import { useUIScale } from './hooks/useUIScale'

/**
 * Inner shell that sits inside AuthProvider so it can access the auth
 * context. Applies global side-effects (UI scale) before rendering routes.
 */
function AppShell() {
  useUIScale()
  return <RouterProvider router={router} />
}

export default function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <AppShell />
      </AuthProvider>
    </ErrorBoundary>
  )
}
