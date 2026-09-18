import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarDays, CheckCircle2, Circle, Clock } from 'lucide-react'

import { getEvents, getTasks } from '../api'
import { buildAgenda, localDayKey, type AgendaDay, type AgendaItem } from '../lib/calendar/agenda'
import { useAuthContext } from '../contexts/AuthContext'
import type { NbEvent, Task } from '../types'

/**
 * «Что дальше» — the agenda.
 *
 * 🔴 The only calendar view the product had nowhere. The month and the day both
 * answer "how is this period shaped"; neither answers "what is next", which is
 * the question a phone is usually taken out to ask
 * (docs/razbor-mobilnyy-kalendar-2026-08-19.md §3, option B).
 *
 * Deliberately a list and not a grid: a grid at 375px is the day view, which
 * already exists and is good. Adding a second grid would duplicate it; a list
 * adds the thing that is missing.
 */

const WINDOW_DAYS = 14

export default function Agenda() {
  const { t, i18n } = useTranslation('common')
  const { user } = useAuthContext()
  // The same fallback the calendar page uses, so the two never disagree
  // about which day an evening event belongs to.
  const timezone = user?.timezone || 'Europe/Moscow'

  const [events, setEvents] = useState<NbEvent[]>([])
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // The window starts at local midnight today: something at 09:00 stays in the
  // agenda all morning rather than vanishing the moment it begins.
  const from = useMemo(() => {
    const d = new Date()
    d.setHours(0, 0, 0, 0)
    return d
  }, [])

  useEffect(() => {
    let cancelled = false
    const to = new Date(from)
    to.setDate(to.getDate() + WINDOW_DAYS)

    Promise.all([getEvents(from.toISOString(), to.toISOString()), getTasks()])
      .then(([evs, tsks]) => {
        if (cancelled) return
        setEvents(evs)
        setTasks(tsks)
      })
      .catch((e: unknown) => {
        if (cancelled) return
        // Said out loud, not swallowed: an empty agenda and a failed load look
        // identical, and the first one is a reason to relax.
        setError(e instanceof Error ? e.message : 'Не удалось загрузить')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [from])

  // 🔴 «Сегодня» rather than the weekday name for the first day. The agenda is
  // read to find out what is next, and «пятница» makes you work out whether
  // that is now — one extra step on every read.
  const todayKey = useMemo(() => localDayKey(new Date(), timezone), [timezone])

  const agenda = useMemo(
    () => buildAgenda(events, tasks, timezone, from, WINDOW_DAYS),
    [events, tasks, timezone, from],
  )

  return (
    <div className="p-3 max-w-2xl mx-auto" data-testid="agenda">
      <h1 className="text-lg font-semibold text-zinc-100 mb-3 flex items-center gap-2">
        <CalendarDays size={18} className="text-blue-400" />
        {t('nav.agenda')}
      </h1>

      {loading && <p className="text-zinc-400 text-sm">{t('agenda.loading')}</p>}

      {error && (
        <p className="text-red-400 text-sm" data-testid="agenda-error">
          {error}
        </p>
      )}

      {!loading && !error && agenda.length === 0 && (
        <p className="text-zinc-400 text-sm" data-testid="agenda-empty">
          {t('agenda.empty', { days: WINDOW_DAYS })}
        </p>
      )}

      <div className="flex flex-col gap-4">
        {agenda.map((day) => (
          <AgendaDaySection
            key={day.key}
            day={day}
            timezone={timezone}
            locale={i18n.language}
            todayKey={todayKey}
          />
        ))}
      </div>
    </div>
  )
}

function AgendaDaySection({
  day,
  timezone,
  locale,
  todayKey,
}: {
  day: AgendaDay
  timezone: string
  locale: string
  todayKey: string
}) {
  const { t } = useTranslation('common')
  // The locale comes from the interface language, not from a hardcoded 'ru-RU':
  // an English user reading «пятница» is the same defect as the notification
  // buttons that stayed Russian.
  const formatted = new Intl.DateTimeFormat(locale, {
    timeZone: timezone,
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  }).format(day.day)
  const heading = day.key === todayKey ? `${t('agenda.today')} · ${formatted}` : formatted

  return (
    <section data-testid="agenda-day" data-day={day.key}>
      <h2 className="text-xs uppercase tracking-wide text-zinc-500 mb-1">{heading}</h2>
      <ul className="flex flex-col gap-1">
        {day.items.map((item) => (
          <AgendaRow key={`${item.kind}-${item.id}`} item={item} timezone={timezone} />
        ))}
      </ul>
    </section>
  )
}

function AgendaRow({ item, timezone }: { item: AgendaItem; timezone: string }) {
  const { t, i18n } = useTranslation('common')
  const time = item.allDay
    ? null
    : new Intl.DateTimeFormat(i18n.language, {
        timeZone: timezone,
        hour: '2-digit',
        minute: '2-digit',
      }).format(item.at)

  return (
    <li
      data-testid="agenda-item"
      data-kind={item.kind}
      className="flex items-start gap-2 rounded border border-zinc-700 bg-zinc-800/60 px-2 py-1.5"
    >
      <span className="mt-0.5 flex-shrink-0 text-zinc-400">
        {item.kind === 'task' ? (
          item.done ? (
            <CheckCircle2 size={14} className="text-green-400" />
          ) : (
            <Circle size={14} />
          )
        ) : (
          <Clock size={14} />
        )}
      </span>

      {/* 🔴 min-w-0 and truncate, not break-words. The same lesson as the week
          grid on 18.09: an unbreakable title at 375px either tows the row wider
          than the screen or wraps one character per line. */}
      <span
        className={`min-w-0 flex-1 truncate text-sm ${
          item.done ? 'text-zinc-500 line-through' : 'text-zinc-100'
        }`}
      >
        {item.title || t('agenda.untitled')}
      </span>

      <span className="flex-shrink-0 text-xs text-zinc-400 tabular-nums">
        {time ?? t('agenda.allDay')}
      </span>
    </li>
  )
}
