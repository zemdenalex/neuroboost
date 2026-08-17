import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CalendarPlus, ListTodo, Timer, Clock } from 'lucide-react'
import { useAuthContext } from '../../contexts/AuthContext'
import { EmptyState } from '../../components/ui/EmptyState'
import { getEvents, getTasks } from '../../api'
import type { NbEvent } from '../../types'
import type { Task } from '../../types'
import { dateLocale } from '../../utils/date'

function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatDate(date: Date, locale: string): string {
  return date.toLocaleDateString(locale, {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  })
}

function isOverdue(task: Task): boolean {
  if (!task.dueDate) return false
  if (task.status === 'DONE' || task.status === 'CANCELLED') return false
  return new Date(task.dueDate) < new Date()
}

export function Dashboard() {
  const { t, i18n } = useTranslation('home')
  const { user } = useAuthContext()

  const [events, setEvents] = useState<NbEvent[]>([])
  const [tasks, setTasks] = useState<Task[]>([])
  const [loadingEvents, setLoadingEvents] = useState(true)
  const [loadingTasks, setLoadingTasks] = useState(true)

  useEffect(() => {
    const today = new Date()
    const todayStart = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 0, 0, 0)
    const todayEnd = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59)

    getEvents(todayStart.toISOString(), todayEnd.toISOString())
      .then(setEvents)
      .catch(() => setEvents([]))
      .finally(() => setLoadingEvents(false))

    getTasks()
      .then(setTasks)
      .catch(() => setTasks([]))
      .finally(() => setLoadingTasks(false))
  }, [])

  const displayName = user?.display_name || user?.tg_first_name || user?.email?.split('@')[0] || 'User'
  const today = new Date()
  const todayLabel = formatDate(today, dateLocale(i18n.language))

  const todayEvents = events
    .slice()
    .sort((a, b) => new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime())
    .slice(0, 6)

  const todoCount = tasks.filter(t => t.status === 'TODO' || t.status === 'IN_PROGRESS' || t.status === 'SCHEDULED').length
  const doneCount = tasks.filter(t => t.status === 'DONE').length
  const overdueCount = tasks.filter(isOverdue).length

  return (
    <div className="h-full overflow-y-auto bg-zinc-950 text-zinc-100">
      <div className="max-w-4xl mx-auto px-4 py-8 space-y-8">
        {/* Greeting */}
        <div>
          <h1 className="text-3xl font-bold">
            {t('dashboard.greeting', { name: displayName })}
          </h1>
          <p className="text-zinc-400 mt-1">{todayLabel}</p>
        </div>

        {/* Quick Actions */}
        <div data-hint="home.quickAdd" className="grid grid-cols-3 gap-4">
          <Link
            to="/calendar"
            className="flex flex-col items-center gap-2 p-4 bg-zinc-900 border border-zinc-800 hover:border-blue-600 rounded-xl transition-colors"
          >
            <CalendarPlus size={24} className="text-blue-400" />
            <span className="text-sm font-medium text-zinc-200">{t('dashboard.newEvent')}</span>
          </Link>
          <Link
            to="/tasks"
            className="flex flex-col items-center gap-2 p-4 bg-zinc-900 border border-zinc-800 hover:border-green-600 rounded-xl transition-colors"
          >
            <ListTodo size={24} className="text-green-400" />
            <span className="text-sm font-medium text-zinc-200">{t('dashboard.newTask')}</span>
          </Link>
          <Link
            to="/tools/pomodoro"
            className="flex flex-col items-center gap-2 p-4 bg-zinc-900 border border-zinc-800 hover:border-red-600 rounded-xl transition-colors"
          >
            <Timer size={24} className="text-red-400" />
            <span className="text-sm font-medium text-zinc-200">{t('dashboard.pomodoro')}</span>
          </Link>
        </div>

        {/* Today's Schedule + Task Summary */}
        <div className="grid md:grid-cols-2 gap-6">
          {/* Today's Schedule */}
          <div data-hint="home.schedule" className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
            <h2 className="font-semibold text-zinc-100 mb-4 flex items-center gap-2">
              <Clock size={18} className="text-blue-400" />
              {t('dashboard.todaySchedule')}
            </h2>

            {loadingEvents ? (
              <div className="flex justify-center py-6">
                <div className="animate-spin rounded-full h-6 w-6 border-t-2 border-b-2 border-blue-500" />
              </div>
            ) : todayEvents.length === 0 ? (
              <EmptyState
                icon={CalendarPlus}
                title={t('dashboard.noEvents')}
                description={t('dashboard.noEventsHint')}
                className="py-6"
              />
            ) : (
              <ul className="space-y-2">
                {todayEvents.map(event => (
                  <li key={event.id} className="flex items-start gap-3">
                    <span className="text-xs text-zinc-500 w-12 shrink-0 pt-0.5 font-mono">
                      {event.allDay ? '——' : formatTime(event.startsAt)}
                    </span>
                    <span className="text-sm text-zinc-200 leading-snug">{event.title}</span>
                  </li>
                ))}
              </ul>
            )}

            <Link
              to="/calendar"
              className="mt-4 block text-xs text-blue-400 hover:text-blue-300 transition-colors text-right"
            >
              {t('dashboard.events')} →
            </Link>
          </div>

          {/* Task Summary */}
          <div data-hint="home.tasks" className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
            <h2 className="font-semibold text-zinc-100 mb-4 flex items-center gap-2">
              <ListTodo size={18} className="text-green-400" />
              {t('dashboard.taskSummary')}
            </h2>

            {loadingTasks ? (
              <div className="flex justify-center py-6">
                <div className="animate-spin rounded-full h-6 w-6 border-t-2 border-b-2 border-green-500" />
              </div>
            ) : tasks.length === 0 ? (
              <EmptyState
                icon={ListTodo}
                title={t('dashboard.noTasks')}
                description={t('dashboard.noTasksHint')}
                className="py-6"
              />
            ) : (
              <div className="grid grid-cols-3 gap-3">
                <div className="bg-zinc-800 rounded-lg p-4 text-center">
                  <div className="text-2xl font-bold text-blue-400">{todoCount}</div>
                  <div className="text-xs text-zinc-400 mt-1">{t('dashboard.todo')}</div>
                </div>
                <div className="bg-zinc-800 rounded-lg p-4 text-center">
                  <div className="text-2xl font-bold text-green-400">{doneCount}</div>
                  <div className="text-xs text-zinc-400 mt-1">{t('dashboard.done')}</div>
                </div>
                <div className="bg-zinc-800 rounded-lg p-4 text-center">
                  <div className="text-2xl font-bold text-red-400">{overdueCount}</div>
                  <div className="text-xs text-zinc-400 mt-1">{t('dashboard.overdue')}</div>
                </div>
              </div>
            )}

            <Link
              to="/tasks"
              className="mt-4 block text-xs text-green-400 hover:text-green-300 transition-colors text-right"
            >
              {t('dashboard.allTasks')} →
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
