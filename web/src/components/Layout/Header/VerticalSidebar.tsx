import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuthContext } from '../../../contexts/AuthContext'
import {
  Calendar,
  CheckSquare,
  LayoutGrid,
  BookOpen,
  Wrench,
  Settings,
  LogOut,
  Shield,
  Home,
} from 'lucide-react'

const navItems = [
  { path: '/home', label: 'Home', icon: Home },
  { path: '/calendar', label: 'Calendar', icon: Calendar },
  { path: '/tasks', label: 'Tasks', icon: CheckSquare },
  { path: '/planning', label: 'Planning', icon: LayoutGrid },
  { path: '/reflections', label: 'Reflections', icon: BookOpen },
  { path: '/tools', label: 'Tools', icon: Wrench },
  { path: '/settings', label: 'Settings', icon: Settings },
]

export default function VerticalSidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuthContext()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const displayName = user?.display_name || user?.email?.split('@')[0] || 'User'
  const initials = displayName.slice(0, 2).toUpperCase()

  return (
    <aside className="fixed left-0 top-0 w-56 h-screen bg-zinc-900 border-r border-zinc-800 flex flex-col">
      {/* Logo */}
      <div className="p-4 border-b border-zinc-800">
        <Link to="/calendar" className="text-lg font-mono font-bold text-white hover:text-blue-400 transition-colors">
          NeuroBoost
        </Link>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 overflow-y-auto">
        <div className="flex flex-col gap-1">
          {navItems.map(({ path, label, icon: Icon }) => {
            const isActive = location.pathname === path
            return (
              <Link
                key={path}
                to={path}
                className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm font-mono transition-colors ${
                  isActive
                    ? 'bg-zinc-800 text-white'
                    : 'text-zinc-400 hover:text-white hover:bg-zinc-800/50'
                }`}
              >
                <Icon className="w-4 h-4" />
                {label}
              </Link>
            )
          })}

          {/* Admin link */}
          {user?.is_admin && (
            <Link
              to="/admin"
              className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm font-mono transition-colors mt-2 ${
                location.pathname === '/admin'
                  ? 'bg-zinc-800 text-white'
                  : 'text-zinc-400 hover:text-white hover:bg-zinc-800/50'
              }`}
            >
              <Shield className="w-4 h-4" />
              Admin
            </Link>
          )}
        </div>
      </nav>

      {/* Profile section */}
      <div className="p-3 border-t border-zinc-800">
        {/* User info */}
        <div className="flex items-center gap-3 px-2 py-2 mb-2">
          {user?.tg_photo_url ? (
            <img
              src={user.tg_photo_url}
              alt={displayName}
              className="w-8 h-8 rounded-full"
            />
          ) : (
            <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center text-xs font-mono text-white">
              {initials}
            </div>
          )}
          <div className="flex-1 min-w-0">
            <p className="text-sm font-mono text-white truncate">{displayName}</p>
            <p className="text-xs text-zinc-500 truncate">{user?.email}</p>
          </div>
        </div>

        {/* Logout button */}
        <button
          onClick={handleLogout}
          className="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-mono text-red-400 hover:bg-zinc-800 transition-colors"
        >
          <LogOut className="w-4 h-4" />
          Sign out
        </button>
      </div>
    </aside>
  )
}