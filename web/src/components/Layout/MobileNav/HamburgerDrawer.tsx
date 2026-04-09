import { useState, useEffect, useRef, useCallback } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Home,
  Calendar,
  CheckSquare,
  LayoutGrid,
  BookOpen,
  Wrench,
  Settings,
  User,
  Menu,
  X,
} from 'lucide-react'

const EDGE_ZONE = 20
const SWIPE_THRESHOLD = 60

export function HamburgerDrawer() {
  const { t } = useTranslation('common')
  const [isOpen, setIsOpen] = useState(false)
  const location = useLocation()

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
  const touchStartX = useRef(0)
  const touchStartY = useRef(0)
  const touchCurrentX = useRef(0)
  const drawerRef = useRef<HTMLDivElement>(null)
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

  // Touch handlers for swipe-open from left edge and swipe-close
  const handleTouchStart = useCallback(
    (e: TouchEvent) => {
      const touch = e.touches[0]
      touchStartX.current = touch.clientX
      touchStartY.current = touch.clientY
      touchCurrentX.current = touch.clientX

      // Start tracking if touching left edge (for opening) or drawer is open (for closing)
      if (touch.clientX <= EDGE_ZONE || isOpen) {
        isDragging.current = true
      }
    },
    [isOpen]
  )

  const handleTouchMove = useCallback(
    (e: TouchEvent) => {
      if (!isDragging.current) return

      const touch = e.touches[0]
      touchCurrentX.current = touch.clientX

      // If vertical movement is greater than horizontal, abandon gesture
      const dy = Math.abs(touch.clientY - touchStartY.current)
      const dx = Math.abs(touch.clientX - touchStartX.current)
      if (dy > dx && dx < 10) {
        isDragging.current = false
        return
      }

      // Update drawer position in real time when closing
      if (isOpen && drawerRef.current) {
        const delta = touchStartX.current - touch.clientX
        if (delta > 0) {
          const translateX = Math.min(delta, 288) // w-72 = 288px
          drawerRef.current.style.transform = `translateX(-${translateX}px)`
          drawerRef.current.style.transition = 'none'
        }
      }
    },
    [isOpen]
  )

  const handleTouchEnd = useCallback(() => {
    if (!isDragging.current) return
    isDragging.current = false

    const dx = touchCurrentX.current - touchStartX.current

    if (!isOpen && dx > SWIPE_THRESHOLD && touchStartX.current <= EDGE_ZONE) {
      // Swipe right from left edge => open
      setIsOpen(true)
    } else if (isOpen && dx < -SWIPE_THRESHOLD) {
      // Swipe left while open => close
      setIsOpen(false)
    }

    // Reset drawer inline styles
    if (drawerRef.current) {
      drawerRef.current.style.transform = ''
      drawerRef.current.style.transition = ''
    }
  }, [isOpen])

  useEffect(() => {
    document.addEventListener('touchstart', handleTouchStart, { passive: true })
    document.addEventListener('touchmove', handleTouchMove, { passive: true })
    document.addEventListener('touchend', handleTouchEnd)

    return () => {
      document.removeEventListener('touchstart', handleTouchStart)
      document.removeEventListener('touchmove', handleTouchMove)
      document.removeEventListener('touchend', handleTouchEnd)
    }
  }, [handleTouchStart, handleTouchMove, handleTouchEnd])

  return (
    <>
      {/* Hamburger trigger button - fixed in header area on mobile */}
      <button
        onClick={() => setIsOpen(true)}
        className="fixed top-2.5 left-3 z-[60] md:hidden p-2 text-zinc-400 hover:text-white rounded-md hover:bg-zinc-800 transition-colors"
        aria-label="Open navigation menu"
      >
        <Menu className="w-5 h-5" />
      </button>

      {/* Overlay backdrop */}
      <div
        className={`fixed inset-0 bg-black/50 z-[70] md:hidden transition-opacity duration-300 ${
          isOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
        onClick={() => setIsOpen(false)}
      />

      {/* Drawer panel */}
      <div
        ref={drawerRef}
        className={`fixed top-0 left-0 w-72 h-full bg-zinc-900 border-r border-zinc-800 z-[80] md:hidden transition-transform duration-300 ${
          isOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        {/* Header with close button */}
        <div className="flex items-center justify-between p-4 border-b border-zinc-800">
          <span className="text-lg font-mono font-bold text-white">NeuroBoost</span>
          <button
            onClick={() => setIsOpen(false)}
            className="p-1 text-zinc-400 hover:text-white rounded-md hover:bg-zinc-800 transition-colors"
            aria-label="Close navigation menu"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Navigation items */}
        <nav className="p-3 overflow-y-auto" style={{ maxHeight: 'calc(100% - 65px)' }}>
          <div className="flex flex-col gap-1">
            {navItems.map(({ path, label, icon: Icon }) => {
              const isActive = location.pathname === path
              return (
                <Link
                  key={path}
                  to={path}
                  className={`flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-mono transition-colors ${
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
          </div>
        </nav>
      </div>
    </>
  )
}
