import { createBrowserRouter, Navigate, Outlet } from 'react-router-dom'
import { useAuthContext } from './contexts/AuthContext'
import { Layout } from './components/Layout'
import { FeedbackButton } from './components/FeedbackButton'

// Pages
import Home from './pages/Home'
import Login from './pages/Login'
import Calendar from './pages/Calendar'
import Tasks from './pages/Tasks'
import Planning from './pages/Planning'
import Reflections from './pages/Reflections'
import Tools from './pages/Tools'
import Settings from './pages/Settings'
import Admin from './pages/Admin'

// Protected route wrapper
function ProtectedRoute({ children }: { children?: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuthContext()

  if (loading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children || <Outlet />}</>
}

// Public route wrapper (redirects to calendar if logged in)
function PublicRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuthContext()

  if (loading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
      </div>
    )
  }

  if (isAuthenticated) {
    return <Navigate to="/calendar" replace />
  }

  return <>{children}</>
}

// Layout with feedback button
function AppLayout() {
  return (
    <>
      <Layout>
        <Outlet />
      </Layout>
      <FeedbackButton />
    </>
  )
}

export const router = createBrowserRouter([
  // Public routes
  {
    path: '/login',
    element: (
      <PublicRoute>
        <Login />
        <FeedbackButton />
      </PublicRoute>
    ),
  },

  // Protected routes with layout
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          {
            path: '/',
            element: <Navigate to="/calendar" replace />,
          },
          {
            path: '/home',
            element: <Home />,
          },
          {
            path: '/calendar',
            element: <Calendar />,
          },
          {
            path: '/tasks',
            element: <Tasks />,
          },
          {
            path: '/planning',
            element: <Planning />,
          },
          {
            path: '/reflections',
            element: <Reflections />,
          },
          {
            path: '/tools',
            element: <Tools />,
          },
          {
            path: '/settings',
            element: <Settings />,
          },
        ],
      },
    ],
  },

  // Admin route (has own layout)
  {
    path: '/admin',
    element: (
      <ProtectedRoute>
        <Admin />
        <FeedbackButton />
      </ProtectedRoute>
    ),
  },

  // Catch all
  {
    path: '*',
    element: <Navigate to="/calendar" replace />,
  },
])
