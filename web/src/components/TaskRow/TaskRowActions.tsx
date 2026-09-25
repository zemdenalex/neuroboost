import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarPlus, Edit2, MoreHorizontal, Trash2, X } from 'lucide-react'
import { SWIPE_ACTIONS_PX, swipeOffset, swipeSettle } from '../../lib/tasks/rowActions'

/**
 * The three ways a task row offers its actions on a phone (lib/tasks/rowActions,
 * Denis 25.09: «all three, customizable in settings»). The desktop keeps its
 * hover icons and never renders these.
 */

export interface RowActionHandlers {
  onSchedule: () => void
  onEdit: () => void
  onDelete: () => void
}

/** `menu`: one «⋯» button and a small menu; Delete set apart in red. */
export function RowActionsMenu({ title, onSchedule, onEdit, onDelete }: RowActionHandlers & { title: string }) {
  const { t } = useTranslation('tasks')
  const { t: tc } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const box = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const outside = (e: PointerEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false)
    }
    const escape = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    document.addEventListener('pointerdown', outside)
    document.addEventListener('keydown', escape)
    return () => {
      document.removeEventListener('pointerdown', outside)
      document.removeEventListener('keydown', escape)
    }
  }, [open])

  const pick = (fn: () => void) => () => {
    setOpen(false)
    fn()
  }

  return (
    <div ref={box} className="relative shrink-0">
      <button
        type="button"
        data-testid="task-row-more"
        aria-label={`${tc('action.more')}: ${title}`}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="p-2 text-zinc-400 hover:text-white rounded"
      >
        <MoreHorizontal className="w-5 h-5" />
      </button>
      {open && (
        <div
          role="menu"
          data-testid="task-row-menu"
          className="absolute right-0 top-full z-30 mt-1 w-56 rounded-md border border-zinc-700 bg-zinc-900 py-1 shadow-lg"
        >
          <MenuItem icon={<CalendarPlus className="w-4 h-4" />} onClick={pick(onSchedule)}>
            {t('schedule')}
          </MenuItem>
          <MenuItem icon={<Edit2 className="w-4 h-4" />} onClick={pick(onEdit)} testId="task-edit">
            {t('editTask')}
          </MenuItem>
          <div className="my-1 border-t border-zinc-800" />
          <MenuItem icon={<Trash2 className="w-4 h-4" />} onClick={pick(onDelete)} danger>
            {tc('action.delete')}
          </MenuItem>
        </div>
      )}
    </div>
  )
}

function MenuItem({
  icon,
  children,
  onClick,
  danger,
  testId,
}: {
  icon: ReactNode
  children: ReactNode
  onClick: () => void
  danger?: boolean
  /** Same testid as the desktop pencil, so specs find Edit on either. */
  testId?: string
}) {
  return (
    <button
      type="button"
      role="menuitem"
      data-testid={testId}
      onClick={onClick}
      className={`flex w-full items-center gap-2 px-3 py-2 text-left text-sm font-mono hover:bg-zinc-800 ${
        danger ? 'text-red-400' : 'text-zinc-200'
      }`}
    >
      {icon}
      {children}
    </button>
  )
}

/**
 * `swipe`: the row slides left under the finger and uncovers Schedule and
 * Delete. A tap on a closed row edits; a tap on an open one closes it.
 * Horizontal only: a mostly vertical move is the page scrolling.
 */
export function SwipeRow({
  children,
  onSchedule,
  onDelete,
  onTap,
}: {
  children: ReactNode
  onSchedule: () => void
  onDelete: () => void
  onTap: () => void
}) {
  const { t } = useTranslation('tasks')
  const { t: tc } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const [offset, setOffset] = useState(0)
  const start = useRef<{ x: number; y: number; horizontal: boolean | null; moved: boolean } | null>(null)
  // The click a browser fires after a swipe must not edit the task.
  const swiped = useRef(false)

  const shown = start.current?.horizontal ? offset : open ? -SWIPE_ACTIONS_PX : 0

  return (
    <div className="relative overflow-hidden" data-testid="task-row-swipe" data-open={open ? 'true' : 'false'}>
      <div className="absolute inset-y-0 right-0 flex" style={{ width: SWIPE_ACTIONS_PX }}>
        <button
          type="button"
          onClick={() => {
            setOpen(false)
            onSchedule()
          }}
          aria-label={t('schedule')}
          tabIndex={open ? 0 : -1}
          className="flex flex-1 items-center justify-center bg-blue-700 text-white"
        >
          <CalendarPlus className="w-5 h-5" />
        </button>
        <button
          type="button"
          onClick={() => {
            setOpen(false)
            onDelete()
          }}
          aria-label={tc('action.delete')}
          tabIndex={open ? 0 : -1}
          className="flex flex-1 items-center justify-center bg-red-700 text-white"
        >
          <Trash2 className="w-5 h-5" />
        </button>
      </div>
      <div
        className="relative bg-zinc-900 transition-transform duration-150 motion-reduce:transition-none"
        style={{ transform: `translateX(${shown}px)`, touchAction: 'pan-y' }}
        onPointerDown={(e) => {
          start.current = { x: e.clientX, y: e.clientY, horizontal: null, moved: false }
          swiped.current = false
        }}
        onPointerMove={(e) => {
          const s = start.current
          if (!s) return
          const dx = e.clientX - s.x
          const dy = e.clientY - s.y
          if (s.horizontal === null && Math.hypot(dx, dy) > 8) s.horizontal = Math.abs(dx) > Math.abs(dy)
          if (s.horizontal) {
            s.moved = true
            setOffset(swipeOffset(dx, open))
          }
        }}
        onPointerUp={() => {
          const s = start.current
          start.current = null
          if (s?.moved) {
            swiped.current = true
            setOpen(swipeSettle(offset))
          }
          // Always re-render: with `open` unchanged the row would keep the
          // last finger offset instead of snapping back.
          setOffset(0)
        }}
        onPointerCancel={() => {
          start.current = null
          setOffset(0)
        }}
        onClickCapture={(e) => {
          if (swiped.current) {
            swiped.current = false
            e.stopPropagation()
            return
          }
          // A tap on an open row only closes it; a tap on a closed row edits,
          // except on its own controls (the done circle, a checkbox).
          if (open) {
            e.stopPropagation()
            setOpen(false)
            return
          }
          const control = (e.target as HTMLElement).closest('button, input, a')
          if (!control) onTap()
        }}
      >
        {children}
      </div>
    </div>
  )
}

/** `card`: a sheet from the bottom with the task's actions. */
export function TaskActionSheet({
  title,
  meta,
  onSchedule,
  onEdit,
  onDelete,
  onClose,
}: RowActionHandlers & { title: string; meta?: ReactNode; onClose: () => void }) {
  const { t } = useTranslation('tasks')
  const { t: tc } = useTranslation('common')
  useEffect(() => {
    const escape = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', escape)
    return () => document.removeEventListener('keydown', escape)
  }, [onClose])
  const pick = (fn: () => void) => () => {
    onClose()
    fn()
  }
  return (
    <div className="fixed inset-0 z-50 md:hidden" role="dialog" aria-modal="true" aria-label={title}>
      <button type="button" aria-label={tc('action.close')} className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div
        data-testid="task-action-sheet"
        className="absolute inset-x-0 bottom-0 rounded-t-xl border-t border-zinc-700 bg-zinc-900 p-4 pb-[calc(1rem+env(safe-area-inset-bottom,0px))]"
      >
        <div className="mb-3 flex items-start gap-3">
          <div className="min-w-0 flex-1">
            <p className="font-mono text-base text-white break-words">{title}</p>
            {meta && <div className="mt-1 text-xs text-zinc-400">{meta}</div>}
          </div>
          <button type="button" onClick={onClose} aria-label={tc('action.close')} className="p-1 text-zinc-400">
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="grid grid-cols-3 gap-2">
          <SheetButton icon={<CalendarPlus className="w-5 h-5" />} onClick={pick(onSchedule)}>
            {t('schedule')}
          </SheetButton>
          <SheetButton icon={<Edit2 className="w-5 h-5" />} onClick={pick(onEdit)}>
            {tc('action.edit')}
          </SheetButton>
          <SheetButton icon={<Trash2 className="w-5 h-5" />} onClick={pick(onDelete)} danger>
            {tc('action.delete')}
          </SheetButton>
        </div>
      </div>
    </div>
  )
}

function SheetButton({ icon, children, onClick, danger }: { icon: ReactNode; children: ReactNode; onClick: () => void; danger?: boolean }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex flex-col items-center gap-1 rounded-lg border border-zinc-700 bg-zinc-800 px-2 py-3 text-xs font-mono ${
        danger ? 'text-red-400' : 'text-zinc-200'
      }`}
    >
      {icon}
      <span className="text-center leading-tight">{children}</span>
    </button>
  )
}
