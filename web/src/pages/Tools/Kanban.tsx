import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Settings, X, GripVertical, Clock, Calendar } from 'lucide-react'
import { listTasks, createTask, updateTask } from '../../api/tasks'
import type { Task, TaskStatus } from '../../api/tasks'
import { PRIORITY_DOT_COLORS } from '../../lib/priority'

// ─── Column definitions ──────────────────────────────────────────────────────

const COLUMN_DEFS = [
  { id: 'INBOX' as const, labelKey: 'kanban.col.inbox', color: 'zinc' },
  { id: 'TODO' as const, labelKey: 'kanban.col.todo', color: 'blue' },
  { id: 'IN_PROGRESS' as const, labelKey: 'kanban.col.inProgress', color: 'yellow' },
  { id: 'SCHEDULED' as const, labelKey: 'kanban.col.scheduled', color: 'purple' },
  { id: 'DONE' as const, labelKey: 'kanban.col.done', color: 'green' },
] as const

type KanbanColumnId = typeof COLUMN_DEFS[number]['id']

// Map KanbanColumnId → TaskStatus accepted by backend
// INBOX is treated as TODO on creation; SCHEDULED is set by backend, but we allow manual drag
const COLUMN_TO_STATUS: Record<KanbanColumnId, TaskStatus> = {
  INBOX: 'TODO',
  TODO: 'TODO',
  IN_PROGRESS: 'IN_PROGRESS',
  SCHEDULED: 'SCHEDULED',
  DONE: 'DONE',
}

// Map TaskStatus → which column to render it in
function statusToColumn(status: TaskStatus): KanbanColumnId {
  switch (status) {
    case 'TODO': return 'TODO'
    case 'IN_PROGRESS': return 'IN_PROGRESS'
    case 'SCHEDULED': return 'SCHEDULED'
    case 'DONE': return 'DONE'
    case 'CANCELLED': return 'DONE'
    default: return 'TODO'
  }
}

// ─── Priority dot ─────────────────────────────────────────────────────────────
// Colors come from lib/priority (shared across Tasks, calendar, sidebar, Eisenhower).

function PriorityDot({ priority }: { priority: number }) {
  const color = PRIORITY_DOT_COLORS[priority] ?? PRIORITY_DOT_COLORS[3]
  return <span className={`inline-block w-2 h-2 rounded-full flex-shrink-0 ${color}`} />
}

// ─── Column header accent ─────────────────────────────────────────────────────

const COLUMN_ACCENT: Record<string, string> = {
  zinc: 'border-zinc-500/30 bg-zinc-800/60',
  blue: 'border-blue-500/30 bg-zinc-800/60',
  yellow: 'border-yellow-500/30 bg-zinc-800/60',
  purple: 'border-purple-500/30 bg-zinc-800/60',
  green: 'border-green-500/30 bg-zinc-800/60',
}

const COLUMN_HEADER_TEXT: Record<string, string> = {
  zinc: 'text-zinc-300',
  blue: 'text-blue-400',
  yellow: 'text-yellow-400',
  purple: 'text-purple-400',
  green: 'text-green-400',
}

const COLUMN_COUNT_BG: Record<string, string> = {
  zinc: 'bg-zinc-700 text-zinc-400',
  blue: 'bg-blue-500/20 text-blue-400',
  yellow: 'bg-yellow-500/20 text-yellow-400',
  purple: 'bg-purple-500/20 text-purple-400',
  green: 'bg-green-500/20 text-green-400',
}

// ─── localStorage helpers ─────────────────────────────────────────────────────

const LS_KEY = 'nb-kanban-columns'

function loadColumnVisibility(): Record<KanbanColumnId, boolean> {
  try {
    const raw = localStorage.getItem(LS_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Record<string, boolean>
      const result = {} as Record<KanbanColumnId, boolean>
      for (const col of COLUMN_DEFS) {
        result[col.id] = parsed[col.id] !== false
      }
      return result
    }
  } catch {
    // ignore
  }
  const defaults = {} as Record<KanbanColumnId, boolean>
  for (const col of COLUMN_DEFS) {
    defaults[col.id] = true
  }
  return defaults
}

function saveColumnVisibility(visibility: Record<KanbanColumnId, boolean>) {
  localStorage.setItem(LS_KEY, JSON.stringify(visibility))
}

// ─── Due date helpers ─────────────────────────────────────────────────────────

function formatDueDate(dueDate: string): string {
  const due = new Date(dueDate)
  const now = new Date()
  const diffDays = Math.ceil((due.getTime() - now.getTime()) / (24 * 60 * 60 * 1000))
  if (diffDays < 0) return `${Math.abs(diffDays)}d late`
  if (diffDays === 0) return 'today'
  if (diffDays === 1) return 'tmrw'
  if (diffDays < 7) return `${diffDays}d`
  return due.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

function getDueDateColor(dueDate: string): string {
  const due = new Date(dueDate)
  const now = new Date()
  const diffDays = Math.ceil((due.getTime() - now.getTime()) / (24 * 60 * 60 * 1000))
  if (diffDays < 0) return 'text-red-400'
  if (diffDays === 0) return 'text-orange-400'
  if (diffDays <= 1) return 'text-yellow-400'
  return 'text-zinc-500'
}

// ─── Task Card ────────────────────────────────────────────────────────────────

interface TaskCardProps {
  task: Task
  onDragStart: (e: React.DragEvent, task: Task) => void
}

function TaskCard({ task, onDragStart }: TaskCardProps) {
  return (
    <div
      draggable
      onDragStart={(e) => onDragStart(e, task)}
      className="group bg-zinc-800 border border-zinc-700/50 rounded-lg px-3 py-2.5 cursor-grab
        hover:border-zinc-600 hover:bg-zinc-750 active:cursor-grabbing
        transition-colors select-none"
    >
      <div className="flex items-start gap-2">
        {/* Drag handle */}
        <GripVertical className="w-3.5 h-3.5 text-zinc-600 group-hover:text-zinc-500 mt-0.5 flex-shrink-0" />

        <div className="flex-1 min-w-0">
          {/* Title + priority dot */}
          <div className="flex items-start gap-1.5">
            <PriorityDot priority={task.priority} />
            <span
              className={`text-sm leading-snug font-medium break-words ${
                task.status === 'DONE' ? 'line-through text-zinc-500' : 'text-zinc-200'
              }`}
            >
              {task.title}
            </span>
          </div>

          {/* Meta row */}
          {(task.estimated_minutes || task.due_date) && (
            <div className="flex items-center gap-2 mt-1.5 text-xs text-zinc-500">
              {task.estimated_minutes && (
                <span className="flex items-center gap-0.5">
                  <Clock className="w-3 h-3" />
                  {task.estimated_minutes}m
                </span>
              )}
              {task.due_date && (
                <span className={`flex items-center gap-0.5 ${getDueDateColor(task.due_date)}`}>
                  <Calendar className="w-3 h-3" />
                  {formatDueDate(task.due_date)}
                </span>
              )}
            </div>
          )}

          {/* Tags */}
          {task.tags.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1.5">
              {task.tags.slice(0, 3).map((tag) => (
                <span key={tag} className="px-1.5 py-px text-[10px] bg-zinc-700/70 text-zinc-400 rounded">
                  {tag}
                </span>
              ))}
              {task.tags.length > 3 && (
                <span className="text-[10px] text-zinc-600">+{task.tags.length - 3}</span>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// ─── Quick Add Input ──────────────────────────────────────────────────────────

interface QuickAddProps {
  onAdd: (title: string) => Promise<void>
  onCancel: () => void
  placeholder: string
}

function QuickAdd({ onAdd, onCancel, placeholder }: QuickAddProps) {
  const [value, setValue] = useState('')
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = value.trim()
    if (!trimmed) return
    setLoading(true)
    try {
      await onAdd(trimmed)
    } finally {
      setLoading(false)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Escape') onCancel()
  }

  return (
    <form onSubmit={handleSubmit} className="mt-2">
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={loading}
        className="w-full bg-zinc-700 border border-zinc-600 rounded-lg px-3 py-2 text-sm text-zinc-100
          placeholder-zinc-500 focus:outline-none focus:border-blue-500 transition-colors"
      />
      <div className="flex gap-2 mt-1.5">
        <button
          type="submit"
          disabled={loading || !value.trim()}
          className="flex-1 text-xs bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed
            text-white rounded-md py-1.5 transition-colors"
        >
          {loading ? '...' : 'Add'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-3 text-xs bg-zinc-700 hover:bg-zinc-600 text-zinc-300 rounded-md py-1.5 transition-colors"
        >
          <X className="w-3 h-3" />
        </button>
      </div>
    </form>
  )
}

// ─── Kanban Column ────────────────────────────────────────────────────────────

interface KanbanColumnProps {
  colDef: typeof COLUMN_DEFS[number]
  label: string
  tasks: Task[]
  addingIn: KanbanColumnId | null
  onStartAdd: (colId: KanbanColumnId) => void
  onCancelAdd: () => void
  onAdd: (title: string, colId: KanbanColumnId) => Promise<void>
  onDragStart: (e: React.DragEvent, task: Task) => void
  onDragOver: (e: React.DragEvent, colId: KanbanColumnId) => void
  onDragLeave: (colId: KanbanColumnId) => void
  onDrop: (e: React.DragEvent, colId: KanbanColumnId) => void
  isDragOver: boolean
  addLabel: string
  noTasksLabel: string
}

function KanbanColumn({
  colDef,
  label,
  tasks,
  addingIn,
  onStartAdd,
  onCancelAdd,
  onAdd,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
  isDragOver,
  addLabel,
  noTasksLabel,
}: KanbanColumnProps) {
  const accentClass = COLUMN_ACCENT[colDef.color]
  const headerText = COLUMN_HEADER_TEXT[colDef.color]
  const countBg = COLUMN_COUNT_BG[colDef.color]

  return (
    <div
      className={`flex flex-col flex-shrink-0 w-[250px] sm:w-[280px] rounded-xl border transition-all duration-150
        ${accentClass}
        ${isDragOver ? 'ring-2 ring-blue-500/50 border-blue-500/50' : ''}`}
      onDragOver={(e) => onDragOver(e, colDef.id)}
      onDragLeave={() => onDragLeave(colDef.id)}
      onDrop={(e) => onDrop(e, colDef.id)}
    >
      {/* Column header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-zinc-700/50">
        <div className="flex items-center gap-2">
          <span className={`text-sm font-semibold ${headerText}`}>
            {label}
          </span>
          <span className={`text-xs px-1.5 py-0.5 rounded-full font-medium ${countBg}`}>
            {tasks.length}
          </span>
        </div>
        <button
          onClick={() => onStartAdd(colDef.id)}
          className="w-6 h-6 rounded-md flex items-center justify-center text-zinc-500
            hover:text-zinc-300 hover:bg-zinc-700 transition-colors"
          title={addLabel}
        >
          <Plus className="w-4 h-4" />
        </button>
      </div>

      {/* Cards */}
      <div className="flex-1 overflow-y-auto p-2 flex flex-col gap-2 min-h-[120px] max-h-[calc(100vh-220px)]">
        {tasks.length === 0 && addingIn !== colDef.id && (
          <p className="text-xs text-zinc-600 text-center mt-4">{noTasksLabel}</p>
        )}
        {tasks.map((task) => (
          <TaskCard key={task.id} task={task} onDragStart={onDragStart} />
        ))}
        {addingIn === colDef.id && (
          <QuickAdd
            onAdd={(title) => onAdd(title, colDef.id)}
            onCancel={onCancelAdd}
            placeholder={addLabel}
          />
        )}
      </div>
    </div>
  )
}

// ─── Settings Panel ───────────────────────────────────────────────────────────

interface SettingsPanelProps {
  visibility: Record<KanbanColumnId, boolean>
  onChange: (colId: KanbanColumnId, visible: boolean) => void
  onClose: () => void
  labels: Record<KanbanColumnId, string>
  title: string
}

function SettingsPanel({ visibility, onChange, onClose, labels, title }: SettingsPanelProps) {
  return (
    <div className="absolute top-12 right-0 z-20 bg-zinc-800 border border-zinc-700 rounded-xl shadow-xl p-4 min-w-[200px]">
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm font-semibold text-zinc-200">{title}</span>
        <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300 transition-colors">
          <X className="w-4 h-4" />
        </button>
      </div>
      <div className="flex flex-col gap-2">
        {COLUMN_DEFS.map((col) => (
          <label key={col.id} className="flex items-center gap-2.5 cursor-pointer group">
            <input
              type="checkbox"
              checked={visibility[col.id]}
              onChange={(e) => onChange(col.id, e.target.checked)}
              className="w-4 h-4 rounded border-zinc-600 bg-zinc-700 accent-blue-500"
            />
            <span className={`text-sm ${COLUMN_HEADER_TEXT[col.color]}`}>{labels[col.id]}</span>
          </label>
        ))}
      </div>
    </div>
  )
}

// ─── Filter Bar ───────────────────────────────────────────────────────────────

interface FilterBarProps {
  priorityFilter: number | null
  onPriorityChange: (p: number | null) => void
}

const PRIORITY_OPTIONS = [1, 2, 3, 4, 5, 0].map((value) => ({
  value,
  label: `P${value}`,
  dotColor: PRIORITY_DOT_COLORS[value],
}))

function FilterBar({ priorityFilter, onPriorityChange }: FilterBarProps) {
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      {PRIORITY_OPTIONS.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onPriorityChange(priorityFilter === opt.value ? null : opt.value)}
          className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-colors
            ${priorityFilter === opt.value
              ? 'bg-zinc-600 text-zinc-100'
              : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-200'
            }`}
        >
          <span className={`w-2 h-2 rounded-full ${opt.dotColor}`} />
          {opt.label}
        </button>
      ))}
    </div>
  )
}

// ─── Kanban Page ──────────────────────────────────────────────────────────────

export default function Kanban() {
  const { t } = useTranslation('tools')

  // Tasks state
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Column visibility
  const [columnVisibility, setColumnVisibility] = useState<Record<KanbanColumnId, boolean>>(loadColumnVisibility)
  const [showSettings, setShowSettings] = useState(false)

  // Drag state
  const [dragOverCol, setDragOverCol] = useState<KanbanColumnId | null>(null)
  const dragTaskRef = useRef<Task | null>(null)

  // Quick add
  const [addingIn, setAddingIn] = useState<KanbanColumnId | null>(null)

  // Filter
  const [priorityFilter, setPriorityFilter] = useState<number | null>(null)

  // Translated column labels
  const colLabels: Record<KanbanColumnId, string> = {
    INBOX: t('kanban.col.inbox'),
    TODO: t('kanban.col.todo'),
    IN_PROGRESS: t('kanban.col.inProgress'),
    SCHEDULED: t('kanban.col.scheduled'),
    DONE: t('kanban.col.done'),
  }

  // ── Load tasks ──────────────────────────────────────────────────────────────
  const loadTasks = useCallback(async () => {
    try {
      setError(null)
      const data = await listTasks()
      setTasks(data)
    } catch {
      setError(t('kanban.error.load'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadTasks()
  }, [loadTasks])

  // ── Column visibility ───────────────────────────────────────────────────────
  function handleVisibilityChange(colId: KanbanColumnId, visible: boolean) {
    const next = { ...columnVisibility, [colId]: visible }
    setColumnVisibility(next)
    saveColumnVisibility(next)
  }

  // ── Drag & Drop ─────────────────────────────────────────────────────────────
  function handleDragStart(e: React.DragEvent, task: Task) {
    dragTaskRef.current = task
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('application/json', JSON.stringify({ taskId: task.id }))
  }

  function handleDragOver(e: React.DragEvent, colId: KanbanColumnId) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    setDragOverCol(colId)
  }

  function handleDragLeave(colId: KanbanColumnId) {
    setDragOverCol((prev) => (prev === colId ? null : prev))
  }

  async function handleDrop(e: React.DragEvent, targetColId: KanbanColumnId) {
    e.preventDefault()
    setDragOverCol(null)

    const task = dragTaskRef.current
    dragTaskRef.current = null
    if (!task) return

    const newStatus = COLUMN_TO_STATUS[targetColId]
    const currentColumn = statusToColumn(task.status)
    if (currentColumn === targetColId) return

    // Optimistic update
    setTasks((prev) =>
      prev.map((t) => (t.id === task.id ? { ...t, status: newStatus } : t))
    )

    try {
      await updateTask(task.id, { status: newStatus })
    } catch {
      // Revert on error
      setTasks((prev) =>
        prev.map((t) => (t.id === task.id ? { ...t, status: task.status } : t))
      )
    }
  }

  // ── Quick Add ───────────────────────────────────────────────────────────────
  async function handleAddTask(title: string, colId: KanbanColumnId) {
    const status = COLUMN_TO_STATUS[colId]
    const newTask = await createTask({ title, status, priority: 3 })
    setTasks((prev) => [newTask, ...prev])
    setAddingIn(null)
  }

  // ── Grouping & filtering ────────────────────────────────────────────────────
  // INBOX is a create-only column — tasks created there get TODO status,
  // but are rendered in the TODO column to avoid duplication.
  function getColumnTasksActual(colId: KanbanColumnId): Task[] {
    if (colId === 'INBOX') {
      return []
    }
    return tasks
      .filter((t) => statusToColumn(t.status) === colId)
      .filter((t) => priorityFilter === null || t.priority === priorityFilter)
  }

  const visibleCols = COLUMN_DEFS.filter((col) => columnVisibility[col.id])

  // ── Render ──────────────────────────────────────────────────────────────────
  return (
    <div className="flex flex-col h-[calc(100vh-56px)] overflow-hidden">
      {/* Top bar */}
      <div className="flex-shrink-0 flex flex-wrap items-center justify-between gap-2 px-4 py-3 border-b border-zinc-800">
        <div className="flex flex-wrap items-center gap-3 min-w-0">
          <h1 className="text-lg font-bold text-zinc-100 shrink-0">{t('kanban.title')}</h1>
          <FilterBar priorityFilter={priorityFilter} onPriorityChange={setPriorityFilter} />
        </div>

        {/* Settings toggle */}
        <div className="relative">
          <button
            onClick={() => setShowSettings((v) => !v)}
            className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm transition-colors
              ${showSettings
                ? 'bg-zinc-700 text-zinc-200'
                : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-200'}`}
          >
            <Settings className="w-4 h-4" />
            <span className="hidden sm:inline">{t('kanban.customizeColumns')}</span>
          </button>

          {showSettings && (
            <SettingsPanel
              visibility={columnVisibility}
              onChange={handleVisibilityChange}
              onClose={() => setShowSettings(false)}
              labels={colLabels}
              title={t('kanban.customizeColumns')}
            />
          )}
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="flex-shrink-0 mx-4 mt-3 px-3 py-2 bg-red-500/10 border border-red-500/30 rounded-lg text-sm text-red-400">
          {error}
        </div>
      )}

      {/* Board */}
      {loading ? (
        <div className="flex-1 flex items-center justify-center">
          <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
        </div>
      ) : (
        <div className="flex-1 overflow-x-auto overflow-y-hidden">
          <div className="flex gap-3 p-4 h-full items-start">
            {visibleCols.map((col) => {
              const colTasks = getColumnTasksActual(col.id)
              return (
                <KanbanColumn
                  key={col.id}
                  colDef={col}
                  label={colLabels[col.id]}
                  tasks={colTasks}
                  addingIn={addingIn}
                  onStartAdd={setAddingIn}
                  onCancelAdd={() => setAddingIn(null)}
                  onAdd={handleAddTask}
                  onDragStart={handleDragStart}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                  isDragOver={dragOverCol === col.id}
                  addLabel={t('kanban.addTask')}
                  noTasksLabel={t('kanban.noTasks')}
                />
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
