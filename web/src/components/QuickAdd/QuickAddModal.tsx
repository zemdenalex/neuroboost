import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { X } from 'lucide-react'
import { QuickAddRow } from './QuickAddRow'
import { createTask, createTasksBatch, type CreateTaskRequest } from '../../api/tasks'

interface QuickAddModalProps {
  open: boolean
  onClose: () => void
}

/**
 * Capture from anywhere. Deliberately the same <QuickAddRow /> as the Tasks
 * page — one component, two mount points, so the two can never behave
 * differently.
 */
export function QuickAddModal({ open, onClose }: QuickAddModalProps) {
  const { t } = useTranslation('tasks')
  const navigate = useNavigate()

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 p-4 pt-24"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={t('quickAdd.placeholder')}
        className="w-full max-w-xl rounded-lg border border-zinc-800 bg-zinc-950 p-4 shadow-xl"
        onClick={e => e.stopPropagation()}
      >
        <div className="mb-3 flex items-center justify-between">
          <span className="font-mono text-sm text-zinc-400">{t('quickAdd.globalTitle')}</span>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('quickAdd.close')}
            className="rounded p-1 text-zinc-500 hover:text-zinc-200"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <QuickAddRow
          autoFocus
          onCreate={(request: CreateTaskRequest) => createTask(request)}
          onCreateMany={requests => createTasksBatch(requests)}
          onOpenFull={() => {
            // The full editor lives on the Tasks page; go there rather than
            // duplicating the whole form inside this overlay.
            onClose()
            navigate('/tasks')
          }}
        />

        <p className="mt-3 font-mono text-xs text-zinc-600">{t('quickAdd.globalHint')}</p>
      </div>
    </div>
  )
}
