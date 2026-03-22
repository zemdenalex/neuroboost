import { useState, useRef, useEffect } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuthContext } from '../../../contexts/AuthContext'
import {
  Calendar,
  CheckSquare,
  LayoutGrid,
  BookOpen,
  Wrench,
  Settings,
  User,
  LogOut,
  Shield,
  ChevronDown,
  Home,
} from 'lucide-react'

export default function HorizontalHeader() {
  const { t } = useTranslation('common')
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuthContext()
  const [isDropdownOpen, setIsDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const navItems = [
    { path: '/home', label: t('nav.home'), icon: Home },
    { path: '/calendar', label: t('nav.calendar'), icon: Calendar },
    { path: '/tasks', label: t('nav.tasks'), icon: CheckSquare },
    { path: '/planning', label: t('nav.planning'), icon: LayoutGrid },
    { path: '/reflections', label: t('nav.reflections'), icon: BookOpen },
    { path: '/tools', label: t('nav.tools'), icon: Wrench },
    { path: '/settings', label: t('nav.settings'), icon: Settings },
    { path: '/profile', label: t('nav.profile'), icon: User },
  ]

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const displayName = user?.display_name || user?.email?.split('@')[0] || 'User'
  const initials = displayName.slice(0, 2).toUpperCase()

  return (
    <header className="fixed top-0 left-0 right-0 bg-zinc-900 border-b border-zinc-800 z-50">
      <div className="px-4 py-2 flex items-center justify-between">
        {/* Logo */}
        <Link to="/calendar" className="text-lg font-mono font-bold text-white hover:text-blue-400 transition-colors">
          NeuroBoost
        </Link>

        {/* Navigation */}
        <nav className="hidden md:flex items-center gap-1">
          {navItems.map(({ path, label, icon: Icon }) => {
            const isActive = location.pathname === path
            return (
              <Link
                key={path}
                to={path}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-mono transition-colors ${
                  isActive
                    ? 'bg-zinc-800 text-white'
                    : 'text-zinc-400 hover:text-white hover:bg-zinc-800/50'
                }`}
              >
                <Icon className="w-4 h-4" />
                <span className="hidden md:inline">{label}</span>
              </Link>
            )
          })}
        </nav>

        {/* Profile Dropdown */}
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={() => setIsDropdownOpen(!isDropdownOpen)}
            className="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-zinc-800 transition-colors"
          >
            {/* Avatar */}
            {user?.tg_photo_url ? (
              <img
                src={user.tg_photo_url}
                alt={displayName}
                className="w-7 h-7 rounded-full"
              />
            ) : (
              <div className="w-7 h-7 rounded-full bg-blue-600 flex items-center justify-center text-xs font-mono text-white">
                {initials}
              </div>
            )}
            <span className="hidden sm:inline text-sm text-zinc-300 font-mono max-w-[120px] truncate">
              {displayName}
            </span>
            <ChevronDown className={`w-4 h-4 text-zinc-400 transition-transform ${isDropdownOpen ? 'rotate-180' : ''}`} />
          </button>

          {/* Dropdown Menu */}
          {isDropdownOpen && (
            <div className="absolute right-0 mt-2 w-56 bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl py-1 z-50">
              {/* User info */}
              <div className="px-3 py-2 border-b border-zinc-800">
                <p className="text-sm font-mono text-white truncate">{displayName}</p>
                <p className="text-xs text-zinc-500 truncate">{user?.email}</p>
              </div>

              {/* Menu items */}
              <Link
                to="/profile"
                onClick={() => setIsDropdownOpen(false)}
                className="flex items-center gap-2 px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-800 transition-colors"
              >
                <User className="w-4 h-4" />
                {t('nav.profile')}
              </Link>

              <Link
                to="/settings"
                onClick={() => setIsDropdownOpen(false)}
                className="flex items-center gap-2 px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-800 transition-colors"
              >
                <Settings className="w-4 h-4" />
                {t('nav.settings')}
              </Link>

              {user?.is_admin && (
                <Link
                  to="/admin"
                  onClick={() => setIsDropdownOpen(false)}
                  className="flex items-center gap-2 px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-800 transition-colors"
                >
                  <Shield className="w-4 h-4" />
                  {t('nav.admin')}
                </Link>
              )}

              <div className="border-t border-zinc-800 mt-1 pt-1">
                <button
                  onClick={handleLogout}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-400 hover:bg-zinc-800 transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                  {t('action.signOut')}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}