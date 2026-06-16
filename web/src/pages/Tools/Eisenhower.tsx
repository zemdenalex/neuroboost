import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Grid2X2, RefreshCw, AlertCircle } from 'lucide-react'
import { getTasks, updateTask } from '../../api'
import type { Task } from '../../types'
import { PRIORITY_DOT_COLORS } from '../../lib/priority'

// ─── Priority → Quadrant mapping ─────────────────────────────────────────────

type QuadrantId = 'q1' | 'q2' | 'q3' | 'q4'

// Priority 1 (Emergency) → Q1 Do First
// Priority 2 (ASAP) → Q1 Do First
// Priority 3 (Normal) → Q2 Schedule
// Priority 4 (Low) → Q3 Delegate
// Priority 5 (If Possible) → Q4 Eliminate
// Priority 0 (Buffer) → Q4 Eliminate
function priorityToQuadrant(priority: number): QuadrantId {
  if (priority === 1 || priority === 2) return 'q1'
  if (priority === 3) return 'q2'
  if (priority === 4) return 'q3'
  return 'q4'
}

// What priority value to assign when dropped into a quadrant
// We pick the most representative priority for each quadrant
const QUADRANT_TO_PRIORITY: Record<QuadrantId, number> = {
  q1: 1,
  q2: 3,
  q3: 4,
  q4: 5,
}

// ─── Quadrant definitions ─────────────────────────────────────────────────────

interface QuadrantDef {
  id: QuadrantId
  titleKey: string
  subtitleKey: string
  borderClass: string
  bgClass: string
  headerClass: string
  dotColor: string
  badgeBg: string
}

const QUADRANT_DEFS: QuadrantDef[] = [
  {
    id: 'q1',
    titleKey: 'eisenhower.q1.title',
    subtitleKey: 'eisenhower.q1.subtitle',
    borderClass: 'border-red-800',
    bgClass: 'bg-red-900/20',
    headerClass: 'text-red-400',
    dotColor: 'bg-red-500',
    badgeBg: 'bg-red-500/20 text-red-400',
  },
  {
    id: 'q2',
    titleKey: 'eisenhower.q2.title',
    subtitleKey: 'eisenhower.q2.subtitle',
    borderClass: 'border-blue-800',
    bgClass: 'bg-blue-900/20',
    headerClass: 'text-blue-400',
    dotColor: 'bg-blue-500',
    badgeBg: 'bg-blue-500/20 text-blue-400',
  },
  {
    id: 'q3',
    titleKey: 'eisenhower.q3.title',
    subtitleKey: 'eisenhower.q3.subtitle',
    borderClass: 'border-yellow-800',
    bgClass: 'bg-yellow-900/20',
    headerClass: 'text-yellow-400',
    dotColor: 'bg-yellow-500',
    badgeBg: 'bg-yellow-500/20 text-yellow-400',
  },
  {
    id: 'q4',
    titleKey: 'eisenhower.q4.title',
    subtitleKey: 'eisenhower.q4.subtitle',
    borderClass: 'border-zinc-700',
    bgClass: 'bg-zinc-800/50',
    headerClass: 'text-zinc-400',
    dotColor: 'bg-zinc-500',
    badgeBg: 'bg-zinc-700 text-zinc-400',
  },
]

// ─── Task Card ────────────────────────────────────────────────────────────────

interface TaskCardProps {
  task: Task
  dotColor: string
  onDragStart: (e: React.DragEvent, task: Task) => void
}

function TaskCard({ task, dotColor, onDragStart }: TaskCardProps) {
  const dotClass = PRIORITY_DOT_COLORS[task.priority] ?? dotColor

  return (
    <div
      draggable
      onDragStart={(e) => onDragStart(e, task)}
      className="group flex items-start gap-2 px-3 py-2 bg-zinc-900 border border-zinc-700/50
        rounded-lg cursor-grab active:cursor-grabbing hover:border-zinc-600 transition-colors select-none"
    >
      <span
        className={`mt-1.5 w-2 h-2 rounded-full flex-shrink-0 ${dotClass}`}
        aria-hidden
      />
      <span className="text-sm text-zinc-200 leading-snug break-words min-w-0 flex-1">
        {task.title}
      </span>
    </div>
  )
}

// ─── Quadrant ─────────────────────────────────────────────────────────────────

interface QuadrantProps {
  def: QuadrantDef
  tasks: Task[]
  isDragOver: boolean
  dragHintKey: string
  onDragStart: (e: React.DragEvent, task: Task) => void
  onDragOver: (e: React.DragEvent, qId: QuadrantId) => void
  onDragLeave: () => void
  onDrop: (e: React.DragEvent, qId: QuadrantId) => void
}

function Quadrant({
  def,
  tasks,
  isDragOver,
  dragHintKey,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
}: QuadrantProps) {
  const { t } = useTranslation('tools')

  return (
    <div
      className={`flex flex-col rounded-xl border transition-all duration-150
        ${def.borderClass} ${def.bgClass}
        ${isDragOver ? 'ring-2 ring-blue-500/40 scale-[1.01]' : ''}`}
      onDragOver={(e) => onDragOver(e, def.id)}
      onDragLeave={onDragLeave}
      onDrop={(e) => onDrop(e, def.id)}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-700/40">
        <div className="flex items-center gap-2">
          <span className={`text-sm font-bold ${def.headerClass}`}>{t(def.titleKey)}</span>
          <span className={`text-xs px-1.5 py-0.5 rounded-full font-medium ${def.badgeBg}`}>
            {tasks.length}
          </span>
        </div>
        <span className="text-xs text-zinc-500 hidden sm:block">{t(def.subtitleKey)}</span>
      </div>

      {/* Task list */}
      <div className="flex-1 flex flex-col gap-2 p-3 min-h-[140px] overflow-y-auto max-h-[calc(50vh-80px)]">
        {tasks.length === 0 ? (
          <p className="text-xs text-zinc-600 text-center mt-4 select-none">
            {isDragOver ? t(dragHintKey) : '—'}
          </p>
        ) : (
          tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              dotColor={def.dotColor}
              onDragStart={onDragStart}
            />
          ))
        )}
        {isDragOver && tasks.length > 0 && (
          <div className="h-1 rounded-full bg-blue-500/30 mx-2" />
        )}
      </div>
    </div>
  )
}

// ─── Axis Labels ──────────────────────────────────────────────────────────────

function AxisLabels() {
  const { t } = useTranslation('tools')

  return (
    <>
      {/* Urgency (horizontal) */}
      <div className="flex items-center justify-between px-1 mb-1 text-xs text-zinc-500 font-mono">
        <span>← {t('eisenhower.axis.notUrgent')}</span>
        <span className="font-semibold text-zinc-400">{t('eisenhower.axis.urgency')}</span>
        <span>{t('eisenhower.axis.urgent')} →</span>
      </div>
    </>
  )
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function Eisenhower() {
  const { t } = useTranslation('tools')

  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [dragOverQuadrant, setDragOverQuadrant] = useState<QuadrantId | null>(null)
  const dragTaskRef = useRef<Task | null>(null)

  // ── Load tasks ──────────────────────────────────────────────────────────────
  const loadTasks = useCallback(async () => {
    try {
      setError(null)
      setLoading(true)
      const data = await getTasks()
      // Only active tasks (not done/cancelled)
      setTasks(data.filter((t) => t.status !== 'DONE' && t.status !== 'CANCELLED'))
    } catch {
      setError(t('eisenhower.error.load'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadTasks()
  }, [loadTasks])

  // ── Group tasks into quadrants ──────────────────────────────────────────────
  function getQuadrantTasks(qId: QuadrantId): Task[] {
    return tasks.filter((t) => priorityToQuadrant(t.priority) === qId)
  }

  // ── Drag handlers ───────────────────────────────────────────────────────────
  function handleDragStart(e: React.DragEvent, task: Task) {
    dragTaskRef.current = task
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', task.id)
  }

  function handleDragOver(e: React.DragEvent, qId: QuadrantId) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    setDragOverQuadrant(qId)
  }

  function handleDragLeave() {
    setDragOverQuadrant(null)
  }

  async function handleDrop(e: React.DragEvent, targetQId: QuadrantId) {
    e.preventDefault()
    setDragOverQuadrant(null)

    const task = dragTaskRef.current
    dragTaskRef.current = null
    if (!task) return

    const newPriority = QUADRANT_TO_PRIORITY[targetQId]
    const currentQId = priorityToQuadrant(task.priority)
    if (currentQId === targetQId) return

    // Optimistic update
    setTasks((prev) =>
      prev.map((t) => (t.id === task.id ? { ...t, priority: newPriority } : t))
    )

    try {
      await updateTask(task.id, { priority: newPriority })
    } catch {
      // Revert on error
      setTasks((prev) =>
        prev.map((t) => (t.id === task.id ? { ...t, priority: task.priority } : t))
      )
    }
  }

  // ── Render ──────────────────────────────────────────────────────────────────
  return (
    <div className="flex flex-col h-[calc(100vh-56px)] overflow-hidden">
      {/* Top bar */}
      <div className="flex-shrink-0 flex items-center justify-between px-4 py-3 border-b border-zinc-800">
        <div className="flex items-center gap-2.5">
          <Grid2X2 className="w-5 h-5 text-yellow-400" />
          <h1 className="text-lg font-bold text-zinc-100">{t('eisenhower.title')}</h1>
        </div>
        <button
          onClick={() => void loadTasks()}
          disabled={loading}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm bg-zinc-800
            text-zinc-400 hover:bg-zinc-700 hover:text-zinc-200 transition-colors disabled:opacity-50"
          title={t('eisenhower.refresh')}
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          <span className="hidden sm:inline">{t('eisenhower.refresh')}</span>
        </button>
      </div>

      {/* Error */}
      {error && (
        <div className="flex-shrink-0 mx-4 mt-3 flex items-center gap-2 px-3 py-2
          bg-red-500/10 border border-red-500/30 rounded-lg text-sm text-red-400">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
        </div>
      )}

      {/* Loading */}
      {loading ? (
        <div className="flex-1 flex items-center justify-center">
          <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto p-4">
          {/* Importance axis label (vertical) */}
          <div className="flex gap-4">
            {/* Left axis */}
            <div className="hidden sm:flex flex-col items-center justify-center w-5 flex-shrink-0 gap-1">
              <span className="text-[10px] text-zinc-500 font-mono rotate-180 [writing-mode:vertical-rl] tracking-widest">
                {t('eisenhower.axis.notImportant')}
              </span>
              <div className="flex-1 w-px bg-zinc-700/50" />
              <span className="text-zinc-400 font-mono text-[10px]">↕</span>
              <div className="flex-1 w-px bg-zinc-700/50" />
              <span className="text-[10px] text-zinc-400 font-mono rotate-180 [writing-mode:vertical-rl] tracking-widest">
                {t('eisenhower.axis.important')}
              </span>
            </div>

            {/* Matrix */}
            <div className="flex-1 flex flex-col gap-2">
              <AxisLabels />

              {/* 2×2 grid: Q2 | Q1 on row 1, Q4 | Q3 on row 2 */}
              {/* Layout: Not Urgent | Urgent (left to right), Important | Not Important (top to bottom) */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {/* Row 1: Important */}
                {/* Q2 — Not Urgent + Important */}
                <Quadrant
                  def={QUADRANT_DEFS[1]}
                  tasks={getQuadrantTasks('q2')}
                  isDragOver={dragOverQuadrant === 'q2'}
                  dragHintKey="eisenhower.dropHere"
                  onDragStart={handleDragStart}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                />
                {/* Q1 — Urgent + Important */}
                <Quadrant
                  def={QUADRANT_DEFS[0]}
                  tasks={getQuadrantTasks('q1')}
                  isDragOver={dragOverQuadrant === 'q1'}
                  dragHintKey="eisenhower.dropHere"
                  onDragStart={handleDragStart}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                />

                {/* Row 2: Not Important */}
                {/* Q4 — Not Urgent + Not Important */}
                <Quadrant
                  def={QUADRANT_DEFS[3]}
                  tasks={getQuadrantTasks('q4')}
                  isDragOver={dragOverQuadrant === 'q4'}
                  dragHintKey="eisenhower.dropHere"
                  onDragStart={handleDragStart}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                />
                {/* Q3 — Urgent + Not Important */}
                <Quadrant
                  def={QUADRANT_DEFS[2]}
                  tasks={getQuadrantTasks('q3')}
                  isDragOver={dragOverQuadrant === 'q3'}
                  dragHintKey="eisenhower.dropHere"
                  onDragStart={handleDragStart}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                />
              </div>

              {/* Drag hint */}
              <p className="text-center text-xs text-zinc-600 mt-2 select-none">
                {t('eisenhower.dragHint')}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
