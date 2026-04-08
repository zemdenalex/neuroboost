import { useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Calendar, CheckSquare } from 'lucide-react'

interface QuickAddDialogProps {
  open: boolean
  onClose: () => void
  anchorRef: React.RefObject<HTMLButtonElement | null>
}

export function QuickAddDialog({ open, onClose, anchorRef }: QuickAddDialogProps) {
  const { t } = useTranslation('common')
  const dialogRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  useEffect(() => {
    if (!open) return

    function handleClickOutside(event: MouseEvent) {
      const target = event.target as Node
      if (
        dialogRef.current &&
        !dialogRef.current.contains(target) &&
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

  const items = [
    { label: t('quickAction.newEvent'), icon: Calendar, path: '/calendar' },
    { label: t('quickAction.newTask'), icon: CheckSquare, path: '/tasks' },
  ]

  return (
    <div
      ref={dialogRef}
      className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-44 bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl overflow-hidden"
    >
      {items.map(({ label, icon: Icon, path }) => (
        <button
          key={path}
          onClick={() => {
            onClose()
            navigate(path)
          }}
          className="w-full flex items-center gap-3 px-4 py-3 text-sm font-mono text-zinc-300 hover:bg-zinc-800 transition-colors"
        >
          <Icon className="w-4 h-4 text-zinc-400" />
          {label}
        </button>
      ))}
    </div>
  )
}
