import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate, Outlet } from 'react-router-dom'
import { useAuthContext } from './contexts/AuthContext'
import { Layout } from './components/Layout'
import { FeedbackButton } from './components/FeedbackButton'

// Lazy-loaded pages
const Home = lazy(() => import('./pages/Home'))
const Login = lazy(() => import('./pages/Login'))
const Calendar = lazy(() => import('./pages/Calendar'))
const Tasks = lazy(() => import('./pages/Tasks'))
const Planning = lazy(() => import('./pages/Planning'))
const Reflections = lazy(() => import('./pages/Reflections'))
const Tools = lazy(() => import('./pages/Tools'))
const Settings = lazy(() => import('./pages/Settings'))
const Admin = lazy(() => import('./pages/Admin'))
const Profile = lazy(() => import('./pages/Profile'))
const Pomodoro = lazy(() => import('./pages/Tools/Pomodoro'))
const Kanban = lazy(() => import('./pages/Tools/Kanban'))

// Suspense fallback
function PageLoader() {
  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
      <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
    </div>
  )
}

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
    return <Navigate to="/home" replace />
  }

  return <>{children}</>
}

// Layout with feedback button and Suspense
function AppLayout() {
  return (
    <>
      <Layout>
        <Suspense fallback={<PageLoader />}>
          <Outlet />
        </Suspense>
      </Layout>
      <FeedbackButton />
    </>
  )
}

export const router = createBrowserRouter([
  // Public routes (no auth required)
  {
    path: '/login',
    element: (
      <PublicRoute>
        <Suspense fallback={<PageLoader />}>
          <Login />
        </Suspense>
        <FeedbackButton />
      </PublicRoute>
    ),
  },

  // Home page — accessible to everyone; Home.tsx handles auth split internally
  {
    path: '/',
    element: <Navigate to="/home" replace />,
  },
  {
    path: '/home',
    element: (
      <>
        <Suspense fallback={<PageLoader />}>
          <Home />
        </Suspense>
        <FeedbackButton />
      </>
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
            path: '/tools/pomodoro',
            element: <Pomodoro />,
          },
          {
            path: '/tools/kanban',
            element: <Kanban />,
          },
          {
            path: '/settings',
            element: <Settings />,
          },
          {
            path: '/profile',
            element: <Profile />,
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
        <Suspense fallback={<PageLoader />}>
          <Admin />
        </Suspense>
        <FeedbackButton />
      </ProtectedRoute>
    ),
  },

  // Catch all
  {
    path: '*',
    element: <Navigate to="/home" replace />,
  },
])
