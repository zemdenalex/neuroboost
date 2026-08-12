import { useState, useEffect, useMemo, useRef } from 'react'
import { onTasksChanged } from '../../lib/quickTask/tasksChanged'
import { sortWithinPriority } from '../../lib/quickTask/sortTasks'
import { useTranslation } from 'react-i18next'
import { createInFlightGuard } from '../../lib/inFlightGuard'
import {
  Plus,
  Loader2,
  CheckCircle,
  Circle,
  Clock,
  Calendar,
  Tag,
  ChevronDown,
  ChevronRight,
  Trash2,
  Edit2,
  Search,
  Filter,
  CalendarPlus,
} from 'lucide-react'
import { QuickAddRow } from '../../components/QuickAdd'
import { showToast } from '../../components/ui/Toast'
import { selectRange } from '../../lib/quickTask/selectRange'
import { startOfLocalDay } from '../../lib/quickTask/localDay'
import {
  listTasks,
  createTask,
  createTasksBatch,
  updateTask,
  deleteTask,
  scheduleTask,
  Task,
  TaskStatus,
  CreateTaskRequest,
  PRIORITY_LABELS,
  PRIORITY_COLORS,
  CONTEXT_ICONS,
} from '../../api/tasks'
import { defaultScheduleSlot } from '../../lib/schedule/defaultScheduleSlot'
import { toDateTimeLocalValue, fromDateTimeLocalValue } from '../../lib/datetime/dateTimeLocal'
import { ReminderOffsets } from '../../components/ReminderOffsets/ReminderOffsets'
import { useReminderSettings } from '../../hooks/useReminderSettings'

export default function Tasks() {
  const { t } = useTranslation('tasks')
  const { t: tc } = useTranslation('common')
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [showEditor, setShowEditor] = useState(false)
  const [editingTask, setEditingTask] = useState<Partial<Task> | null>(null)
  const [saving, setSaving] = useState(false)
  const [editorError, setEditorError] = useState<string | null>(null)
  const reminderSettings = useReminderSettings()
  // Stable across renders so a synchronous double-click is blocked (a useState
  // flag would not update before the second click's handler reads it).
  const saveGuard = useRef(createInFlightGuard()).current
  // Separate guard so a double-tap on Schedule can't POST twice — the backend
  // INSERTs a new event per call (no upsert), so re-entrancy duplicates events.
  const scheduleGuard = useRef(createInFlightGuard()).current
  const [search, setSearch] = useState('')
  const [filterStatus, setFilterStatus] = useState<TaskStatus | 'ALL'>('ALL')
  const [expandedGroups, setExpandedGroups] = useState<Set<number>>(new Set([1, 2, 3, 4, 5]))
  const [selected, setSelected] = useState<Set<string>>(new Set())
  // Anchor for Shift+click range selection — the last row clicked without Shift.
  const selectionAnchor = useRef<string | null>(null)

  // Fetch tasks, and refetch whenever something outside this page creates one
  // (the global Ctrl+K modal). Reloading rather than splicing in the response
  // keeps one source of truth: the modal can create several tasks at once.
  useEffect(() => {
    let cancelled = false
    const fetchTasks = async (showSpinner: boolean) => {
      if (showSpinner) setLoading(true)
      try {
        const data = await listTasks()
        if (!cancelled) setTasks(data)
      } catch (error) {
        console.error('Failed to fetch tasks:', error)
      } finally {
        if (showSpinner && !cancelled) setLoading(false)
      }
    }
    fetchTasks(true)
    // No spinner on the refetch: the list is already on screen and flashing it
    // to a loading state would be more disruptive than the update itself.
    const off = onTasksChanged(() => fetchTasks(false))
    return () => { cancelled = true; off() }
  }, [])

  // A filter or search change hides rows; keeping them selected would let a
  // bulk action hit tasks Denis cannot see.
  useEffect(() => {
    setSelected(new Set())
    selectionAnchor.current = null
  }, [filterStatus, search])

  // Clear any stale editor error whenever the editor opens or closes, so a
  // previous failure never lingers on the next open.
  useEffect(() => {
    setEditorError(null)
  }, [showEditor])

  // Filter and group tasks
  const filteredTasks = useMemo(() => {
    return tasks.filter(task => {
      if (filterStatus !== 'ALL' && task.status !== filterStatus) return false
      if (search && !task.title.toLowerCase().includes(search.toLowerCase())) return false
      return true
    })
  }, [tasks, filterStatus, search])

  const tasksByPriority = useMemo(() => {
    const groups = new Map<number, Task[]>()
    for (let p = 1; p <= 5; p++) groups.set(p, [])
    groups.set(0, []) // Buffer
    
    for (const task of filteredTasks) {
      const priority = task.priority ?? 3
      if (groups.has(priority)) {
        groups.get(priority)!.push(task)
      }
    }

    // Order inside a group was previously whatever the API returned, which at
    // fifty tasks a day meant a list that reshuffled under the reader.
    for (const [priority, list] of groups) {
      groups.set(priority, sortWithinPriority(list))
    }

    return groups
  }, [filteredTasks])

  // Handlers
  const toggleGroup = (priority: number) => {
    setExpandedGroups(prev => {
      const next = new Set(prev)
      if (next.has(priority)) {
        next.delete(priority)
      } else {
        next.add(priority)
      }
      return next
    })
  }

  // Closing a task is the most repeated action of the day, so it is optimistic:
  // the checkbox must not wait on the network. A failure rolls the row back.
  const setStatus = async (task: Task, next: TaskStatus) => {
    const previous = task.status
    setTasks(prev => prev.map(t => (t.id === task.id ? { ...t, status: next } : t)))
    try {
      const updated = await updateTask(task.id, { status: next })
      setTasks(prev => prev.map(t => (t.id === updated.id ? updated : t)))
      return true
    } catch (error) {
      console.error('Failed to update task:', error)
      setTasks(prev => prev.map(t => (t.id === task.id ? { ...t, status: previous } : t)))
      return false
    }
  }

  const handleStatusToggle = async (task: Task) => {
    const previous = task.status
    const next: TaskStatus = previous === 'DONE' ? 'TODO' : 'DONE'
    const ok = await setStatus(task, next)
    if (ok && next === 'DONE') {
      showToast(t('toast.closed', { title: task.title }), {
        label: t('toast.undo'),
        onClick: () => void setStatus({ ...task, status: next }, previous),
      })
    }
  }

  // Tap-friendly alternative to drag-scheduling: drop the task on the calendar
  // at a sensible default slot. Optimistically reflect the SCHEDULED status.
  const handleScheduleTask = (task: Task) => {
    const slot = defaultScheduleSlot(new Date(), task.estimated_minutes)
    return scheduleGuard(async () => {
      try {
        await scheduleTask(task.id, { starts_at: slot.startsAt, ends_at: slot.endsAt, all_day: false })
        setTasks(prev => prev.map(t => t.id === task.id ? { ...t, status: 'SCHEDULED' } : t))
      } catch (error) {
        console.error('Failed to schedule task:', error)
      }
    })
  }

  const handleSaveTask = () => {
    if (!editingTask?.title) return
    // Capture the narrowed title: property narrowing from the guard above is not
    // preserved inside the deferred async closure (TS widens it back to string | undefined).
    const title = editingTask.title

    return saveGuard(async () => {
      setSaving(true)
      setEditorError(null)
      try {
        if (editingTask.id) {
          // Update existing
          const updated = await updateTask(editingTask.id, {
            title,
            description: editingTask.description,
            priority: editingTask.priority,
            due_date: editingTask.due_date,
            estimated_minutes: editingTask.estimated_minutes,
            contexts: editingTask.contexts,
            tags: editingTask.tags,
            reminder_offsets: editingTask.reminder_offsets,
          })
          setTasks(prev => prev.map(t => t.id === updated.id ? updated : t))
        } else {
          // Create new
          const created = await createTask({
            title,
            description: editingTask.description,
            priority: editingTask.priority ?? 3,
            due_date: editingTask.due_date,
            estimated_minutes: editingTask.estimated_minutes,
            contexts: editingTask.contexts ?? [],
            tags: editingTask.tags ?? [],
            // Omitted entirely when untouched, so the backend applies the
            // user's default preset. An explicit [] means "deliberately none".
            reminder_offsets: editingTask.reminder_offsets,
          })
          setTasks(prev => [...prev, created])
        }
        setShowEditor(false)
        setEditingTask(null)
      } catch (error) {
        console.error('Failed to save task:', error)
        setEditorError(t('error.saveFailed'))
      } finally {
        setSaving(false)
      }
    })
  }

  const handleDeleteTask = async () => {
    if (!editingTask?.id) return
    
    if (confirm(t('confirmDelete', { title: editingTask.title }))) {
      setEditorError(null)
      try {
        await deleteTask(editingTask.id)
        setTasks(prev => prev.filter(t => t.id !== editingTask.id))
        // A deleted id left in the selection makes the bulk bar count rows
        // that no longer exist.
        setSelected(prev => {
          const next = new Set(prev)
          next.delete(editingTask.id!)
          return next
        })
        setShowEditor(false)
        setEditingTask(null)
      } catch (error) {
        console.error('Failed to delete task:', error)
        setEditorError(t('error.deleteFailed'))
      }
    }
  }

  // Quick-add path: no modal, no reload — the created task is prepended so the
  // list reflects it before the next title is typed.
  async function handleQuickCreate(request: CreateTaskRequest): Promise<Task> {
    const created = await createTask(request)
    setTasks(prev => [created, ...prev])
    return created
  }

  // Multi-line paste: one request for the whole list, rows failing independently.
  async function handleQuickCreateMany(requests: CreateTaskRequest[]) {
    const result = await createTasksBatch(requests)
    setTasks(prev => [...result.tasks, ...prev])
    return result
  }

  // Rows in the order they are rendered, so a Shift+click range matches what
  // is on screen rather than the order the data happens to be in.
  const visibleIds = useMemo(
    () => Array.from(tasksByPriority.values()).flat().map(task => task.id),
    [tasksByPriority],
  )

  const handleRowSelect = (taskId: string, shiftKey: boolean) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (shiftKey && selectionAnchor.current) {
        for (const id of selectRange(visibleIds, selectionAnchor.current, taskId)) next.add(id)
        return next
      }
      if (next.has(taskId)) next.delete(taskId)
      else next.add(taskId)
      selectionAnchor.current = taskId
      return next
    })
  }

  const selectedTasks = () => tasks.filter(task => selected.has(task.id))

  const handleBulkClose = async () => {
    const batch = selectedTasks().filter(task => task.status !== 'DONE')
    setSelected(new Set())
    await Promise.all(batch.map(task => setStatus(task, 'DONE')))
    if (batch.length > 0) showToast(t('toast.bulkClosed', { count: batch.length }))
  }

  const handleBulkTomorrow = async () => {
    const batch = selectedTasks()
    setSelected(new Set())
    const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone
    const due = startOfLocalDay(new Date(), timeZone, 1).toISOString()
    await Promise.all(
      batch.map(async task => {
        try {
          const updated = await updateTask(task.id, { due_date: due })
          setTasks(prev => prev.map(t => (t.id === updated.id ? updated : t)))
        } catch (error) {
          console.error('Failed to move task:', error)
        }
      }),
    )
    if (batch.length > 0) showToast(t('toast.bulkMoved', { count: batch.length }))
  }

  const stats = useMemo(() => ({
    total: tasks.length,
    done: tasks.filter(t => t.status === 'DONE').length,
    todo: tasks.filter(t => t.status === 'TODO').length,
    overdue: tasks.filter(t => t.due_date && new Date(t.due_date) < new Date() && t.status !== 'DONE').length,
  }), [tasks])

  return (
    <div className="flex flex-col h-full">
        {/* Header */}
        <div className="px-6 py-4 border-b border-zinc-800 bg-zinc-900">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-mono font-semibold text-white">{t('title')}</h1>
            <button
              data-hint="tasks.new"
              onClick={() => {
                setEditingTask({ title: '', priority: 3, contexts: [], tags: [] })
                setShowEditor(true)
              }}
              className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-mono rounded-lg transition-colors"
            >
              <Plus className="w-4 h-4" />
              {t('newTask')}
            </button>
          </div>

          {/* Stats */}
          <div className="flex items-center gap-6 text-sm">
            <span className="text-zinc-400">
              {t('stat.total')} <strong className="text-white">{stats.total}</strong>
            </span>
            <span className="text-zinc-400">
              {t('stat.done')} <strong className="text-green-400">{stats.done}</strong>
            </span>
            <span className="text-zinc-400">
              {t('stat.todo')} <strong className="text-blue-400">{stats.todo}</strong>
            </span>
            {stats.overdue > 0 && (
              <span className="text-zinc-400">
                {t('stat.overdue')} <strong className="text-red-400">{stats.overdue}</strong>
              </span>
            )}
          </div>

          {/* Filters */}
          {/* Wraps below sm: the search field and the status select together
              overflowed 375px, clipping the search placeholder to "Search ta". */}
          <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 mt-4">
            <div className="relative flex-1 sm:max-w-md">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('searchPlaceholder')}
                className="w-full pl-10 pr-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              />
            </div>
            
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-zinc-500" />
              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value as TaskStatus | 'ALL')}
                className="px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              >
                <option value="ALL">{t('filter.allStatus')}</option>
                <option value="TODO">{t('filter.todo')}</option>
                <option value="IN_PROGRESS">{t('filter.inProgress')}</option>
                <option value="SCHEDULED">{t('filter.scheduled')}</option>
                <option value="DONE">{t('filter.done')}</option>
              </select>
            </div>
          </div>
        </div>

        {/* Task List */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {selected.size > 0 && (
            <div className="flex items-center gap-3 rounded-lg border border-blue-900 bg-blue-950/40 px-4 py-2">
              <span className="font-mono text-sm text-blue-200">{t('bulk.count', { count: selected.size })}</span>
              <button
                type="button"
                onClick={() => void handleBulkClose()}
                className="rounded border border-blue-800 px-3 py-1 font-mono text-sm text-blue-100 hover:border-blue-500"
              >
                {t('bulk.close')}
              </button>
              <button
                type="button"
                onClick={() => void handleBulkTomorrow()}
                className="rounded border border-blue-800 px-3 py-1 font-mono text-sm text-blue-100 hover:border-blue-500"
              >
                {t('bulk.tomorrow')}
              </button>
              <button
                type="button"
                onClick={() => setSelected(new Set())}
                className="ml-auto font-mono text-sm text-zinc-400 hover:text-zinc-100"
              >
                {t('bulk.clear')}
              </button>
            </div>
          )}
          <QuickAddRow
            autoFocus
            onCreate={handleQuickCreate}
            onCreateMany={handleQuickCreateMany}
            onOpenFull={(draft) => {
              setEditingTask({ contexts: [], tags: [], ...draft })
              setShowEditor(true)
            }}
          />
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 text-zinc-400 animate-spin" />
            </div>
          ) : filteredTasks.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-zinc-500">{t('noTasks')}</p>
              <button
                onClick={() => {
                  setEditingTask({ title: '', priority: 3, contexts: [], tags: [] })
                  setShowEditor(true)
                }}
                className="mt-4 text-blue-400 hover:text-blue-300"
              >
                {t('createFirst')}
              </button>
            </div>
          ) : (
            // Priority groups
            Array.from(tasksByPriority.entries())
              .filter(([, tasks]) => tasks.length > 0)
              .map(([priority, priorityTasks]) => (
                <div key={priority} className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
                  {/* Group header */}
                  <button
                    onClick={() => toggleGroup(priority)}
                    className="w-full flex items-center justify-between px-4 py-3 hover:bg-zinc-800/50 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      {expandedGroups.has(priority) ? (
                        <ChevronDown className="w-4 h-4 text-zinc-500" />
                      ) : (
                        <ChevronRight className="w-4 h-4 text-zinc-500" />
                      )}
                      <div className={`w-3 h-3 rounded-full ${PRIORITY_COLORS[priority]}`} />
                      <span className="font-mono font-medium text-white">
                        {t(`priority.${priority}`)}
                      </span>
                      <span className="text-sm text-zinc-500">({priorityTasks.length})</span>
                    </div>
                  </button>

                  {/* Tasks */}
                  {expandedGroups.has(priority) && (
                    <div className="border-t border-zinc-800">
                      {priorityTasks.map(task => (
                        <div
                          key={task.id}
                          className={`flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-b-0 transition-colors group ${selected.has(task.id) ? 'bg-blue-950/40' : 'hover:bg-zinc-800/30'}`}
                        >
                          {/* Selection — Shift+click extends from the last plain click.
                              Hidden at rest so the row shows ONE control and the
                              "done" circle is unambiguous; it fades in on hover, on
                              keyboard focus, and stays up for every row while a
                              selection exists.
                              Opacity rather than `hidden`: a display:none checkbox
                              leaves the Tab order, which would make bulk selection
                              unreachable without a mouse. */}
                          <input
                            type="checkbox"
                            checked={selected.has(task.id)}
                            onChange={() => {}}
                            onClick={(e) => handleRowSelect(task.id, e.shiftKey)}
                            aria-label={t('bulk.select', { title: task.title })}
                            className={`shrink-0 accent-blue-500 transition-opacity focus-visible:opacity-100 ${
                              selected.size > 0 ? 'opacity-100' : 'opacity-0 group-hover:opacity-100 group-focus-within:opacity-100'
                            }`}
                          />
                          {/* Status toggle */}
                          <button
                            data-hint="tasks.complete"
                            onClick={() => handleStatusToggle(task)}
                            className="shrink-0"
                          >
                            {task.status === 'DONE' ? (
                              <CheckCircle className="w-5 h-5 text-green-500" />
                            ) : (
                              <Circle className="w-5 h-5 text-zinc-600 hover:text-zinc-400" />
                            )}
                          </button>

                          {/* Task content */}
                          <div className="flex-1 min-w-0">
                            <div className={`font-mono text-sm ${task.status === 'DONE' ? 'text-zinc-500 line-through' : 'text-white'}`}>
                              {task.title}
                            </div>
                            
                            {/* Meta info */}
                            <div className="flex items-center gap-3 mt-1">
                              {task.due_date && (
                                <span className={`flex items-center gap-1 text-xs ${
                                  new Date(task.due_date) < new Date() && task.status !== 'DONE'
                                    ? 'text-red-400'
                                    : 'text-zinc-500'
                                }`}>
                                  <Calendar className="w-3 h-3" />
                                  {new Date(task.due_date).toLocaleDateString()}
                                </span>
                              )}
                              {task.estimated_minutes && (
                                <span className="flex items-center gap-1 text-xs text-zinc-500">
                                  <Clock className="w-3 h-3" />
                                  {task.estimated_minutes}m
                                </span>
                              )}
                              {task.contexts?.map(ctx => (
                                <span key={ctx} className="text-xs text-zinc-500">
                                  {CONTEXT_ICONS[ctx] || ''} {ctx}
                                </span>
                              ))}
                              {task.tags?.map(tag => (
                                <span key={tag} className="flex items-center gap-1 text-xs text-zinc-500">
                                  <Tag className="w-3 h-3" />
                                  {tag}
                                </span>
                              ))}
                            </div>
                          </div>

                          {/* Actions — always visible on touch (no hover), hover-revealed on desktop */}
                          <div className="flex items-center gap-1 opacity-100 md:opacity-0 md:group-hover:opacity-100 transition-opacity">
                            <button
                              data-hint="tasks.schedule"
                              onClick={() => handleScheduleTask(task)}
                              title={t('schedule')}
                              aria-label={t('schedule')}
                              className="p-1.5 text-zinc-500 hover:text-blue-400 hover:bg-zinc-700 rounded transition-colors"
                            >
                              <CalendarPlus className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => {
                                setEditingTask(task)
                                setShowEditor(true)
                              }}
                              // Icon-only, like its neighbours — but unlike them
                              // it carried no accessible name at all, so screen
                              // readers announced it as an unlabelled button.
                              title={t('editTask')}
                              aria-label={t('editTask')}
                              className="p-1.5 text-zinc-500 hover:text-white hover:bg-zinc-700 rounded transition-colors"
                            >
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={async () => {
                                if (confirm(t('confirmDelete', { title: task.title }))) {
                                  try {
                                    await deleteTask(task.id)
                                    setTasks(prev => prev.filter(t => t.id !== task.id))
                                  } catch (error) {
                                    // Keep the row (setTasks is skipped on throw) and avoid an
                                    // unhandled rejection. List-level error UI is a follow-up.
                                    console.error('Failed to delete task:', error)
                                  }
                                }
                              }}
                              className="p-1.5 text-zinc-500 hover:text-red-400 hover:bg-zinc-700 rounded transition-colors"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))
          )}
        </div>

        {/* Task Editor Modal */}
        {showEditor && editingTask && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
            <div className="bg-zinc-900 border border-zinc-700 rounded-lg w-full max-w-lg p-6 space-y-4 max-h-[90vh] overflow-y-auto">
              <h2 className="text-lg font-mono font-semibold text-white">
                {editingTask.id ? t('editTask') : t('newTask')}
              </h2>

              <div>
                <label className="block text-sm text-zinc-400 mb-1">{t('form.title')}</label>
                <input
                  type="text"
                  value={editingTask.title || ''}
                  onChange={(e) => setEditingTask(prev => ({ ...prev!, title: e.target.value }))}
                  placeholder={t('form.titlePlaceholder')}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                  autoFocus
                />
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-1">{t('form.description')}</label>
                <textarea
                  value={editingTask.description || ''}
                  onChange={(e) => setEditingTask(prev => ({ ...prev!, description: e.target.value }))}
                  placeholder={t('form.descriptionPlaceholder')}
                  rows={3}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500 resize-none"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-zinc-400 mb-1">{t('form.priority')}</label>
                  <select
                    value={editingTask.priority ?? 3}
                    onChange={(e) => setEditingTask(prev => ({ ...prev!, priority: Number(e.target.value) }))}
                    className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                  >
                    {Object.keys(PRIORITY_LABELS).map((value) => (
                      <option key={value} value={value}>{t(`priority.${value}`)}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm text-zinc-400 mb-1">{t('form.dueDate')}</label>
                  <input
                    type="datetime-local"
                    value={editingTask.due_date ? toDateTimeLocalValue(editingTask.due_date) : ''}
                    onChange={(e) => setEditingTask(prev => ({ ...prev!, due_date: e.target.value ? fromDateTimeLocalValue(e.target.value) : undefined }))}
                    className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-1">{t('form.estimatedTime')}</label>
                <input
                  type="number"
                  value={editingTask.estimated_minutes || ''}
                  onChange={(e) => setEditingTask(prev => ({ ...prev!, estimated_minutes: Number(e.target.value) || undefined }))}
                  placeholder={t('form.estimatedPlaceholder')}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>

              {/* Reminders count back from due_date, so without one there is
                  nothing to count from — the control says so rather than
                  silently accepting offsets that could never fire. Same rule as
                  quick-add's second level, which until now was the ONLY place a
                  task's reminders could be set: once created, they could never
                  be changed again. */}
              <div>
                <ReminderOffsets
                  value={editingTask.reminder_offsets ?? []}
                  onChange={offsets => setEditingTask(prev => ({ ...prev!, reminder_offsets: offsets }))}
                  presets={reminderSettings.presets}
                  disabled={!editingTask.due_date}
                />
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-2">{t('form.contexts')}</label>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(CONTEXT_ICONS).map(([ctx, icon]) => (
                    <button
                      key={ctx}
                      onClick={() => {
                        const contexts = editingTask.contexts || []
                        if (contexts.includes(ctx)) {
                          setEditingTask(prev => ({ ...prev!, contexts: contexts.filter(c => c !== ctx) }))
                        } else {
                          setEditingTask(prev => ({ ...prev!, contexts: [...contexts, ctx] }))
                        }
                      }}
                      className={`px-3 py-1.5 rounded-lg text-sm font-mono transition-colors ${
                        editingTask.contexts?.includes(ctx)
                          ? 'bg-blue-600 text-white'
                          : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
                      }`}
                    >
                      {icon} {ctx}
                    </button>
                  ))}
                </div>
              </div>

              {editorError && (
                <div
                  role="alert"
                  className="p-3 bg-red-900/30 border border-red-800 rounded-lg text-red-400 text-sm"
                >
                  {editorError}
                </div>
              )}

              <div className="flex justify-between pt-4">
                {editingTask.id && (
                  <button
                    onClick={handleDeleteTask}
                    className="px-4 py-2 text-red-400 hover:bg-red-900/30 rounded-lg transition-colors"
                  >
                    {tc('action.delete')}
                  </button>
                )}
                <div className="flex gap-2 ml-auto">
                  <button
                    onClick={() => {
                      setShowEditor(false)
                      setEditingTask(null)
                    }}
                    className="px-4 py-2 text-zinc-400 hover:bg-zinc-800 rounded-lg transition-colors"
                  >
                    {tc('action.cancel')}
                  </button>
                  <button
                    onClick={handleSaveTask}
                    disabled={!editingTask.title || saving}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg transition-colors"
                  >
                    {tc('action.save')}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
    </div>
  )
}