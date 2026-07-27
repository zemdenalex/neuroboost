import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate, Outlet } from 'react-router-dom'
import { useAuthContext } from './contexts/AuthContext'
import { Layout } from './components/Layout'
import { QuickAddModal } from './components/QuickAdd/QuickAddModal'
import { useGlobalQuickAdd } from './hooks/useGlobalQuickAdd'
import { FeedbackButton } from './components/FeedbackButton'
import { PomodoroWidget } from './components/Pomodoro/PomodoroWidget'
import { PomodoroToasts } from './components/Pomodoro/PomodoroToasts'

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
const Eisenhower = lazy(() => import('./pages/Tools/Eisenhower'))
const TimeBlocking = lazy(() => import('./pages/Tools/TimeBlocking'))

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

// Public route wrapper (redirects to /home if logged in)
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
  const quickAdd = useGlobalQuickAdd()
  return (
    <>
      <Layout>
        <Suspense fallback={<PageLoader />}>
          <Outlet />
        </Suspense>
      </Layout>
      <PomodoroWidget />
      <PomodoroToasts />
      <FeedbackButton />
      <QuickAddModal open={quickAdd.open} onClose={quickAdd.close} />
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

  // Landing page — unauthenticated root, full-screen without header
  {
    path: '/',
    element: (
      <PublicRoute>
        <Suspense fallback={<PageLoader />}>
          <Home />
        </Suspense>
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
          // Home/Dashboard for authenticated users — inside Layout so header/nav renders
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
            path: '/tools/pomodoro',
            element: <Pomodoro />,
          },
          {
            path: '/tools/kanban',
            element: <Kanban />,
          },
          {
            path: '/tools/eisenhower',
            element: <Eisenhower />,
          },
          {
            path: '/tools/time-blocking',
            element: <TimeBlocking />,
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
