import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight, Pencil, Plus, X, Check, Square, CheckSquare } from 'lucide-react'
import { useAuthContext } from '../../contexts/AuthContext'
import { listTasks, type Task } from '../../api/tasks'
import {
  listDays,
  getProposal,
  confirmDay,
  addDayTask,
  removeDayTask,
  markDayTaskDone,
  type Day,
  type DayItem,
} from '../../api/dayTasks'
import { dayLevelSquare, todayInZone } from '../../lib/dayTasks/dayColour'
import { readDayPrefs, shiftDay, dayWhen, canRemove, hourInZone, errorKey } from '../../lib/dayTasks/dayView'
import { dateLocale } from '../../utils/date'

/**
 * 📌 Day tasks, the bot's screen on the web
 * (spec docs/superpowers/specs/2026-09-24-web-day-tasks-design.md §1).
 * The rules live in lib/dayTasks; this file only lays them out.
 */
export default function DayTasks() {
  const { t, i18n } = useTranslation('daytasks')
  const { user, updateSettings } = useAuthContext()
  const tz = user?.timezone || 'Europe/Moscow'
  const prefs = readDayPrefs(user?.settings)
  const today = todayInZone(new Date(), tz)

  const [day, setDay] = useState(today)
  const [data, setData] = useState<Day | null>(null)
  const [proposal, setProposal] = useState<DayItem[]>([])
  const [tasks, setTasks] = useState<Task[]>([])
  const [editing, setEditing] = useState(false)
  const [picking, setPicking] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const when = dayWhen(day, today)

  const load = useCallback(async () => {
    setError('')
    try {
      const [days, open] = await Promise.all([listDays(day, day), listTasks()])
      const d = days[0] ?? null
      setData(d)
      setTasks(open)
      setProposal(d && !d.confirmed && dayWhen(day, today) !== 'past' ? await getProposal(day) : [])
    } catch (err) {
      setError(t(errorKey(err)))
    }
  }, [day, today, t])

  useEffect(() => {
    if (prefs.enabled) void load()
  }, [load, prefs.enabled])

  // Every write answers with the same shape of error and reloads the day.
  const act = async (fn: () => Promise<unknown>) => {
    setBusy(true)
    setError('')
    try {
      await fn()
      await load()
    } catch (err) {
      setError(t(errorKey(err)))
    } finally {
      setBusy(false)
    }
  }

  const byId = useMemo(() => new Map(tasks.map((x) => [x.id, x])), [tasks])
  const addable = useMemo(() => {
    const inDay = new Set((data?.items ?? []).map((i) => i.task_id))
    return tasks.filter((x) => x.status !== 'DONE' && x.status !== 'CANCELLED' && !inDay.has(x.id))
  }, [tasks, data])

  const label = new Date(`${day}T12:00:00Z`).toLocaleDateString(dateLocale(i18n.language), {
    weekday: 'short',
    day: '2-digit',
    month: '2-digit',
    timeZone: 'UTC',
  })

  if (!prefs.enabled) {
    return (
      <div className="max-w-xl mx-auto p-4">
        <h1 className="text-xl font-mono font-semibold text-white mb-4">📌 {t('title')}</h1>
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <p className="text-zinc-300">{t('off')}</p>
          <button
            onClick={() => void updateSettings({ day_tasks_enabled: true })}
            className="mt-3 px-4 py-2 rounded-md bg-blue-600 text-onaccent text-sm hover:bg-blue-500"
          >
            {t('turnOn')}
          </button>
        </div>
      </div>
    )
  }

  const taken = data?.confirmed === true
  const removable = canRemove(day, today, hourInZone(new Date(), tz))

  return (
    <div className="max-w-xl mx-auto p-4">
      <h1 className="text-xl font-mono font-semibold text-white">📌 {t('title')}</h1>
      <p className="text-sm text-zinc-500 mt-1 mb-4">{t('intro')}</p>

      <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
        <div className="flex items-center justify-between gap-2">
          <button
            aria-label={t('prev')}
            onClick={() => { setDay(shiftDay(day, -1)); setEditing(false); setPicking(false) }}
            className="p-2 rounded-md text-zinc-400 hover:bg-zinc-800"
          >
            <ChevronLeft className="w-5 h-5" />
          </button>
          <div className="text-center">
            <div className="text-white font-mono">{label}</div>
            {day !== today && (
              <button onClick={() => { setDay(today); setEditing(false) }} className="text-xs text-blue-400 hover:underline">
                {t('today')}
              </button>
            )}
          </div>
          <button
            aria-label={t('next')}
            onClick={() => { setDay(shiftDay(day, 1)); setEditing(false); setPicking(false) }}
            className="p-2 rounded-md text-zinc-400 hover:bg-zinc-800"
          >
            <ChevronRight className="w-5 h-5" />
          </button>
        </div>

        {!data && !error && <p className="text-zinc-500 text-sm mt-4">{t('loading')}</p>}

        {data && taken && (
          <div className="mt-4 text-lg font-mono text-white" data-testid="day-progress">
            {dayLevelSquare(data.level)} {t('progress', { done: data.done, target: data.target })}
          </div>
        )}

        {data && !taken && when === 'past' && <p className="text-zinc-400 mt-4">⬛ {t('notTakenPast')}</p>}

        {data && !taken && when !== 'past' && (
          <div className="mt-4">
            <p className="text-zinc-300">{proposal.length ? t('notTaken') : t('proposalEmpty')}</p>
            <ul className="mt-3 space-y-1">
              {proposal.map((p) => (
                <li key={p.task_id} className="text-sm text-zinc-300">⬜ {p.title}</li>
              ))}
            </ul>
            <button
              disabled={busy}
              onClick={() => void act(() => confirmDay(day, proposal.map((p) => p.task_id)))}
              className="mt-3 px-4 py-2 rounded-md bg-blue-600 text-onaccent text-sm hover:bg-blue-500 disabled:opacity-50"
            >
              ✅ {proposal.length ? t('take') : t('takeEmpty')}
            </button>
          </div>
        )}

        {data && taken && (
          <ul className="mt-3 space-y-1">
            {data.items.map((item) => (
              <li key={item.task_id} className="flex items-center gap-2">
                <button
                  disabled={busy || item.done || when !== 'today'}
                  aria-label={t('markDone')}
                  onClick={() => void act(() => markDayTaskDone(byId.get(item.task_id) ?? { id: item.task_id }, day))}
                  className="flex-1 flex items-center gap-2 p-2 rounded-md text-left text-sm text-zinc-200 hover:bg-zinc-800 disabled:hover:bg-transparent"
                >
                  {item.done ? <CheckSquare className="w-4 h-4 text-green-500" /> : <Square className="w-4 h-4 text-zinc-500" />}
                  <span className={item.done ? 'line-through text-zinc-500' : ''}>{item.title}</span>
                </button>
                {editing && removable && (
                  <button
                    aria-label={t('remove')}
                    disabled={busy}
                    onClick={() => void act(() => removeDayTask(day, item.task_id))}
                    className="p-2 rounded-md text-zinc-500 hover:text-red-400 hover:bg-zinc-800"
                  >
                    <X className="w-4 h-4" />
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}

        {data && taken && when === 'past' && <p className="text-xs text-zinc-500 mt-3">{t('pastReadOnly')}</p>}

        {data && taken && when !== 'past' && (
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              onClick={() => { setEditing(!editing); setPicking(false) }}
              className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-zinc-800 text-zinc-300 text-sm hover:bg-zinc-700"
            >
              {editing ? <Check className="w-4 h-4" /> : <Pencil className="w-4 h-4" />}
              {editing ? t('doneEditing') : t('edit')}
            </button>
            {editing && (
              <button
                onClick={() => setPicking(!picking)}
                className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-zinc-800 text-zinc-300 text-sm hover:bg-zinc-700"
              >
                <Plus className="w-4 h-4" /> {t('add')}
              </button>
            )}
          </div>
        )}

        {picking && (
          <div className="mt-3 border-t border-zinc-800 pt-3">
            <div className="text-xs text-zinc-500 mb-2">{t('addPick')}</div>
            <ul className="max-h-64 overflow-y-auto space-y-1">
              {addable.map((x) => (
                <li key={x.id}>
                  <button
                    disabled={busy}
                    onClick={() => void act(() => addDayTask(day, x.id))}
                    className="w-full text-left p-2 rounded-md text-sm text-zinc-300 hover:bg-zinc-800"
                  >
                    {x.title}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}

        {error && <p role="alert" className="mt-3 text-sm text-red-400">{error}</p>}
      </section>
    </div>
  )
}
