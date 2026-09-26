import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CalendarPlus, ListTodo, Timer, Clock, Pin, Circle, CheckCircle2 } from 'lucide-react'
import { useAuthContext } from '../../contexts/AuthContext'
import { EmptyState } from '../../components/ui/EmptyState'
import { getEvents, getTasks, updateTask } from '../../api'
import { markOccurrence } from '../../api/tasks'
import { listDays, type Day } from '../../api/dayTasks'
import { readDayPrefs } from '../../lib/dayTasks/dayView'
import { todayInZone } from '../../lib/dayTasks/dayColour'
import { homeDayLine, homeTasks } from '../../lib/home/todayView'
import { sidebarTick, sidebarTicked } from '../../components/TaskSidebar/sidebarTick'
import { PriorityMark } from '../../components/PriorityMark'
import type { NbEvent } from '../../types'
import type { Task } from '../../types'
import { dateLocale } from '../../utils/date'
import { taskCounts } from '../../lib/home/taskCounts'

function formatTime(iso: string, timeZone: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', timeZone })
}

function formatDate(date: Date, locale: string): string {
  return date.toLocaleDateString(locale, {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  })
}

export function Dashboard() {
  const { t, i18n } = useTranslation('home')
  const { user } = useAuthContext()

  const [events, setEvents] = useState<NbEvent[]>([])
  const [tasks, setTasks] = useState<Task[]>([])
  const [loadingEvents, setLoadingEvents] = useState(true)
  const [loadingTasks, setLoadingTasks] = useState(true)
  const [day, setDay] = useState<Day | undefined>(undefined)
  const timeZone = user?.timezone || 'Europe/Moscow'
  const dayTasksOn = readDayPrefs(user?.settings).enabled

  const loadTasks = useCallback(() => getTasks().then(setTasks).catch(() => setTasks([])), [])

  // The day-tasks line, as the bot's «Сегодня» (gap list row 8). Off: no line.
  useEffect(() => {
    if (!dayTasksOn) return
    const today = todayInZone(new Date(), timeZone)
    listDays(today, today).then((days) => setDay(days[0])).catch(() => setDay(undefined))
  }, [dayTasksOn, timeZone])

  useEffect(() => {
    const today = new Date()
    const todayStart = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 0, 0, 0)
    const todayEnd = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59)

    getEvents(todayStart.toISOString(), todayEnd.toISOString())
      .then(setEvents)
      .catch(() => setEvents([]))
      .finally(() => setLoadingEvents(false))

    loadTasks().finally(() => setLoadingTasks(false))
  }, [loadTasks])

  // A tick answers today for a series and toggles a one-off (lib/tasks/tickAction).
  const tick = async (task: Task) => {
    const action = sidebarTick(task)
    try {
      if (action.kind === 'occurrence') await markOccurrence(task.id, action.state, todayInZone(new Date(), timeZone))
      else await updateTask(task.id, { status: action.next })
    } finally {
      void loadTasks()
    }
  }

  const displayName = user?.display_name || user?.tg_first_name || user?.email?.split('@')[0] || 'User'
  const today = new Date()
  const todayLabel = formatDate(today, dateLocale(i18n.language))

  const todayEvents = events
    .slice()
    .sort((a, b) => new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime())
  const { shown: shownTasks, more: moreTasks } = homeTasks(tasks)
  const dayLine = dayTasksOn ? homeDayLine(day) : null

  // Repeating tasks by their day, not their status (lib/home/taskCounts)
  const { todo: todoCount, done: doneCount, overdue: overdueCount } = taskCounts(tasks)

  return (
    <div className="h-full overflow-y-auto bg-zinc-950 text-zinc-100">
      <div className="max-w-4xl mx-auto px-4 py-8 space-y-8">
        {/* Greeting */}
        <div>
          <h1 className="text-3xl font-bold">
            {t('dashboard.greeting', { name: displayName })}
          </h1>
          <p className="text-zinc-400 mt-1">{todayLabel}</p>
          {dayLine && (
            <div data-testid="home-day-line" className="mt-3 flex flex-wrap items-center gap-3">
              <span className="font-mono text-sm text-zinc-200">
                📌 {dayLine.kind === 'taken'
                  ? t('dashboard.dayLine', { square: dayLine.square, done: dayLine.done, target: dayLine.target })
                  : t('dashboard.dayNotTaken')}
              </span>
              <Link to="/day-tasks" className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-700 bg-zinc-900 text-xs font-mono text-zinc-200 hover:border-zinc-500">
                <Pin size={14} />
                {t('dashboard.dayTasks')}
              </Link>
            </div>
          )}
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
                    <span className="text-xs text-zinc-500 w-24 shrink-0 pt-0.5 font-mono">
                      {event.allDay
                        ? t('dashboard.allDay')
                        : `${formatTime(event.startsAt, timeZone)}–${formatTime(event.endsAt, timeZone)}`}
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
              <>
              {/* The bot's «Сегодня»: five open tasks, most urgent first, then «и ещё N». */}
              <ul data-testid="home-tasks" className="mb-4 space-y-1">
                {shownTasks.map((task) => (
                  <li key={task.id} className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => void tick(task)}
                      aria-label={t('dashboard.tick', { title: task.title })}
                      className="p-1 text-zinc-500 hover:text-green-400"
                    >
                      {sidebarTicked(task) ? <CheckCircle2 size={18} className="text-green-400" /> : <Circle size={18} />}
                    </button>
                    <PriorityMark priority={task.priority} />
                    <Link to={`/tasks?task=${task.id}`} className="min-w-0 flex-1 truncate text-sm text-zinc-200 hover:text-white">
                      {task.title}
                    </Link>
                  </li>
                ))}
                {moreTasks > 0 && (
                  <li className="pl-9 text-xs text-zinc-500">{t('dashboard.more', { count: moreTasks })}</li>
                )}
              </ul>
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
              </>
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
