import { useRef, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { LayoutGrid, Wrench, BookOpen, User } from 'lucide-react'

interface MoreMenuProps {
  open: boolean
  onClose: () => void
  anchorRef: React.RefObject<HTMLButtonElement | null>
}

const moreItems = [
  { path: '/planning', label: 'Planning', icon: LayoutGrid },
  { path: '/tools', label: 'Tools', icon: Wrench },
  { path: '/reflections', label: 'Reflections', icon: BookOpen },
  { path: '/profile', label: 'Profile', icon: User },
]

export function MoreMenu({ open, onClose, anchorRef }: MoreMenuProps) {
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
