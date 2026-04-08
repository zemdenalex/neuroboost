import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Plus,
  Calendar,
  CheckSquare,
  Home,
  LayoutGrid,
  BookOpen,
  Wrench,
  Settings,
  User,
} from 'lucide-react'

const SWIPE_DISMISS_THRESHOLD = 80

export function FabBottomSheet() {
  const { t } = useTranslation('common')
  const [isOpen, setIsOpen] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()

  const quickActions = [
    { label: t('quickAction.newEvent'), icon: Calendar, path: '/calendar' },
    { label: t('quickAction.newTask'), icon: CheckSquare, path: '/tasks' },
  ]

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
  const sheetRef = useRef<HTMLDivElement>(null)
  const dragStartY = useRef(0)
  const isDragging = useRef(false)

  // Close on route change
  useEffect(() => {
    setIsOpen(false)
  }, [location.pathname])

  // Prevent body scroll when open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => {
      document.body.style.overflow = ''
    }
  }, [isOpen])

  // Drag handle: swipe down to dismiss
  const handleDragStart = useCallback((e: React.TouchEvent) => {
    dragStartY.current = e.touches[0].clientY
    isDragging.current = true
  }, [])

  const handleDragMove = useCallback((e: React.TouchEvent) => {
    if (!isDragging.current || !sheetRef.current) return

    const dy = e.touches[0].clientY - dragStartY.current
    if (dy > 0) {
      sheetRef.current.style.transform = `translateY(${dy}px)`
      sheetRef.current.style.transition = 'none'
    }
  }, [])

  const handleDragEnd = useCallback((e: React.TouchEvent) => {
    if (!isDragging.current || !sheetRef.current) return
    isDragging.current = false

    const dy = e.changedTouches[0].clientY - dragStartY.current
    if (dy > SWIPE_DISMISS_THRESHOLD) {
      setIsOpen(false)
    }

    // Reset inline styles
    sheetRef.current.style.transform = ''
    sheetRef.current.style.transition = ''
  }, [])

  const handleNavigate = (path: string) => {
    setIsOpen(false)
    navigate(path)
  }

  return (
    <>
      {/* FAB button */}
      <button
        onClick={() => setIsOpen(true)}
        className={`fixed bottom-20 right-6 w-14 h-14 rounded-full bg-blue-600 hover:bg-blue-500 shadow-lg flex items-center justify-center z-50 md:hidden transition-all duration-300 ${
          isOpen ? 'scale-0 opacity-0' : 'scale-100 opacity-100'
        }`}
        aria-label="Open navigation"
      >
        <Plus className="w-6 h-6 text-white" />
      </button>

      {/* Overlay backdrop */}
      <div
        className={`fixed inset-0 bg-black/50 z-[70] md:hidden transition-opacity duration-300 ${
          isOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
        onClick={() => setIsOpen(false)}
      />

      {/* Bottom sheet */}
      <div
        ref={sheetRef}
        className={`fixed bottom-0 left-0 right-0 bg-zinc-900 rounded-t-2xl border-t border-zinc-800 z-[80] md:hidden transition-transform duration-300 ${
          isOpen ? 'translate-y-0' : 'translate-y-full'
        }`}
      >
        {/* Drag handle */}
        <div
          className="flex justify-center pt-3 pb-2 cursor-grab"
          onTouchStart={handleDragStart}
          onTouchMove={handleDragMove}
          onTouchEnd={handleDragEnd}
        >
          <div className="w-12 h-1 bg-zinc-700 rounded-full" />
        </div>

        {/* Quick actions */}
        <div className="px-4 pb-3">
          <p className="text-xs font-mono text-zinc-500 uppercase tracking-wider mb-2 px-1">
            {t('quickAction.quickActions')}
          </p>
          <div className="flex gap-3">
            {quickActions.map(({ label, icon: Icon, path }) => (
              <button
                key={path}
                onClick={() => handleNavigate(path)}
                className="flex-1 flex items-center gap-2 px-4 py-3 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
              >
                <Icon className="w-4 h-4 text-blue-400" />
                <span className="text-sm font-mono text-zinc-200">{label}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Divider */}
        <div className="border-t border-zinc-800 mx-4" />

        {/* Navigation items */}
        <nav className="px-4 py-3 max-h-[50vh] overflow-y-auto">
          <p className="text-xs font-mono text-zinc-500 uppercase tracking-wider mb-2 px-1">
            {t('quickAction.navigation')}
          </p>
          <div className="flex flex-col gap-0.5">
            {navItems.map(({ path, label, icon: Icon }) => {
              const isActive = location.pathname === path
              return (
                <button
                  key={path}
                  onClick={() => handleNavigate(path)}
                  className={`flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-mono transition-colors text-left ${
                    isActive
                      ? 'bg-zinc-800 text-white'
                      : 'text-zinc-400 hover:text-white hover:bg-zinc-800/50'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  {label}
                </button>
              )
            })}
          </div>
        </nav>

        {/* Bottom safe area padding */}
        <div className="h-6" />
      </div>
    </>
  )
}
