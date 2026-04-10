import { useRef, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LayoutGrid, Wrench, BookOpen, User } from 'lucide-react'
import { useFeatureFlags } from '../../../hooks/useFeatureFlags'

interface MoreMenuProps {
  open: boolean
  onClose: () => void
  anchorRef: React.RefObject<HTMLButtonElement | null>
}

export function MoreMenu({ open, onClose, anchorRef }: MoreMenuProps) {
  const { t } = useTranslation('common')
  const flags = useFeatureFlags()

  const moreItems = [
    { path: '/planning', label: t('nav.planning'), icon: LayoutGrid, enabled: true },
    { path: '/tools', label: t('nav.tools'), icon: Wrench, enabled: flags.tools },
    { path: '/reflections', label: t('nav.reflections'), icon: BookOpen, enabled: true },
    { path: '/profile', label: t('nav.profile'), icon: User, enabled: true },
  ].filter((item) => item.enabled)
  const menuRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (!open) return

    function handleClickOutside(event: MouseEvent) {
      const target = event.target as Node
      if (
        menuRef.current &&
        !menuRef.current.contains(target) &&
        anchorRef.current &&
        !anchorRef.current.contains(target)
      ) {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open, onClose, anchorRef])

  if (!open) return null

  return (
    <div
      ref={menuRef}
      className="absolute bottom-full right-0 mb-2 w-48 bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl overflow-hidden"
    >
      {moreItems.map(({ path, label, icon: Icon }) => {
        const isActive = location.pathname === path
        return (
          <button
            key={path}
            onClick={() => {
              onClose()
              navigate(path)
            }}
            className={`w-full flex items-center gap-3 px-4 py-3 text-sm font-mono transition-colors ${
              isActive
                ? 'bg-zinc-800 text-blue-400'
                : 'text-zinc-300 hover:bg-zinc-800'
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        )
      })}
    </div>
  )
}
