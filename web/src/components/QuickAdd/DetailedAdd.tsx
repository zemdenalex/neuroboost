import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthContext } from '../../contexts/AuthContext'
import { parseLine, type ParsedLine } from '../../api/parse'
import { createEvent } from '../../api/events'
import { createTask } from '../../api/tasks'
import { buildQuickTask } from '../../lib/quickTask/buildQuickTask'
import { resolveQuickTaskSettings } from '../../lib/quickTask/settings'
import { readNewEventLength } from '../../lib/calendar/newEventLength'
import { localTimeOn } from '../../lib/calendar/shiftDays'
import { todayInZone } from '../../lib/dayTasks/dayColour'
import { useOpenDay } from '../../lib/calendar/openDay'
import { announceCreated } from '../../lib/calendar/createdSignal'
import { CalendarPicker } from '../Calendars/CalendarPicker'
import { showToast } from '../ui/Toast'

type Pop = 'date' | 'time' | 'prio' | 'cal' | 'more' | null
type Kind = 'event' | 'task'

const DURATIONS = [15, 30, 60, 90] as const
/** 1 = Emergency … 5 = If possible; 0 = Buffer (gotcha 4: smaller is higher). */
const PRIORITIES = [1, 2, 3, 4, 5, 0] as const
const PARSE_DEBOUNCE_MS = 300

const hhmm = (m: number) => `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`

/** Minutes of the day an instant falls on, in a zone. */
function minutesIn(iso: string, timeZone: string): number {
  const p = new Intl.DateTimeFormat('en-GB', { timeZone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).formatToParts(new Date(iso))
  return Number(p.find((x) => x.type === 'hour')?.value ?? 0) * 60 + Number(p.find((x) => x.type === 'minute')?.value ?? 0)
}

interface Chips { day: string; time: number | null; dur: number; calendarId: string }

/**
 * The chips after a parse of the line: day, time and length from what it
 * names, the calendar it names; a chip set by hand keeps its value, and a time
 * deleted from the line clears the time chip.
 */
function chipsFrom(line: ParsedLine, cur: Chips, manual: { time: boolean; cal: boolean }): Chips {
  const next = { ...cur }
  if (line.kind === 'event' && line.starts_at && line.ends_at && !line.all_day) {
    next.day = todayInZone(new Date(line.starts_at), line.timezone)
    if (!manual.time) {
      next.time = minutesIn(line.starts_at, line.timezone)
      next.dur = Math.max(5, Math.round((Date.parse(line.ends_at) - Date.parse(line.starts_at)) / 60_000))
    }
  } else {
    if (line.due_date) next.day = todayInZone(new Date(line.due_date), line.timezone)
    if (!line.has_time && !manual.time) next.time = null
  }
  if (line.calendar_id && !manual.cal) next.calendarId = line.calendar_id
  return next
}

/**
 * The detailed «+», «like the one in ticktick» (Denis 31.10; his screenshots
 * ref/konkurenty-skriny-2026-10-26/photo_10, 13, 14): title, description and a
 * row of chips. The day chip starts on the day the calendar shows. The line is
 * still read as typed — «завтра 10:00 врач» moves the day and time chips — and
 * the chips can then be changed by hand. A time makes an event, none a task;
 * «⋯» overrides that.
 */
export function DetailedAdd({ initial, onDone }: { initial: string; onDone: () => void }) {
  const { t, i18n } = useTranslation('common')
  const { t: tt } = useTranslation('tasks')
  const { user } = useAuthContext()
  const zone = user?.timezone || 'Europe/Moscow'
  const openDay = useOpenDay()
  const [title, setTitle] = useState(initial)
  // The last parse, with the text it read: used only while that is still the text.
  const [parsed, setParsed] = useState<{ text: string; line: ParsedLine } | null>(null)
  // A time, length or calendar set by hand is not overwritten by the next parse.
  const manual = useRef({ time: false, cal: false })
  const [desc, setDesc] = useState('')
  const [day, setDay] = useState(() => openDay ?? todayInZone(new Date(), zone))
  const [time, setTime] = useState<number | null>(null)
  const [dur, setDur] = useState<number>(() => readNewEventLength(user?.settings))
  const [priority, setPriority] = useState<number | null>(null)
  const [calendarId, setCalendarId] = useState('')
  const [kindOverride, setKindOverride] = useState<Kind | null>(null)
  const [pop, setPop] = useState<Pop>(null)
  const [busy, setBusy] = useState(false)
  const titleRef = useRef<HTMLInputElement>(null)
  const dayRef = useRef(day)
  dayRef.current = day

  useEffect(() => { titleRef.current?.focus() }, [])

  // The typed line sets the chips it names, as TickTick's inline syntax does.
  useEffect(() => {
    const typed = title.trim()
    if (!typed) { setParsed(null); return }
    let cancelled = false
    const timer = setTimeout(() => {
      void parseLine(typed, undefined, dayRef.current).then((line) => {
        if (cancelled || !line) return
        setParsed({ text: typed, line })
        applyRef.current(line)
      })
    }, PARSE_DEBOUNCE_MS)
    return () => { cancelled = true; clearTimeout(timer) }
  }, [title])

  /** Moves the chips to what the line says; a hand-set chip stays. Returns the result for ✓. */
  function applyLine(line: ParsedLine): Chips {
    // From a ref, not this render's values: a parse lands 300 ms after the
    // keystroke and must not undo a chip changed by hand in between.
    const next = chipsFrom(line, chipsRef.current, manual.current)
    setDay(next.day)
    setTime(next.time)
    setDur(next.dur)
    setCalendarId(next.calendarId)
    return next
  }

  const chipsRef = useRef<Chips>({ day, time, dur, calendarId })
  chipsRef.current = { day, time, dur, calendarId }
  const applyRef = useRef(applyLine)
  applyRef.current = applyLine

  const kind: Kind = kindOverride ?? (time !== null ? 'event' : 'task')
  const dayLabel = new Intl.DateTimeFormat(i18n.language, { timeZone: 'UTC', weekday: 'short', day: 'numeric', month: 'short' })
    .format(new Date(day + 'T12:00:00Z'))
  const isToday = day === todayInZone(new Date(), zone)

  async function submit() {
    const typed = title.trim()
    if (!typed || busy) return
    setBusy(true)
    try {
      // ✓ before the parse caught up with the text: read it now, so the saved
      // title never belongs to an earlier version of the line.
      let line = parsed && parsed.text === typed ? parsed.line : null
      let chips: Chips = { day, time, dur, calendarId }
      if (!line) {
        line = await parseLine(typed, undefined, day)
        if (line) chips = applyLine(line)
      }
      const { day: onDay, time: at, dur: length, calendarId: cal } = chips
      const name = (line?.title || typed).trim()
      const kind: Kind = kindOverride ?? (at !== null ? 'event' : 'task')
      const isToday = onDay === todayInZone(new Date(), zone)
      if (kind === 'event') {
        // An event with no time chosen goes on the next half hour of today,
        // or the working day's start on any other day.
        const start = at ?? (isToday ? Math.min(23 * 60 + 30, Math.ceil((minutesIn(new Date().toISOString(), zone) + 1) / 30) * 30) : 9 * 60)
        const startsAt = localTimeOn(onDay, hhmm(start), zone)
        await createEvent({
          title: name,
          description: desc.trim() || undefined,
          starts_at: startsAt,
          ends_at: new Date(Date.parse(startsAt) + length * 60_000).toISOString(),
          calendar_id: cal || undefined,
          // What the line said that has no chip: repeat, reminders, tags.
          ...(line?.rrule ? { rrule: line.rrule } : {}),
          ...(line?.reminder_offsets ? { reminder_offsets: line.reminder_offsets } : {}),
          ...(line?.tags.length ? { tags: line.tags } : {}),
        })
      } else {
        const built = buildQuickTask({ title: name, settings: resolveQuickTaskSettings(user?.settings), now: new Date(), timeZone: zone })
        if (!built) return
        await createTask({
          ...built,
          title: name,
          description: desc.trim() || undefined,
          due_date: localTimeOn(onDay, '00:00', zone),
          ...(priority !== null ? { priority } : {}),
          calendar_id: cal || undefined,
        })
      }
      showToast(t(kind === 'event' ? 'addSheet.createdEventShort' : 'addSheet.createdTask', { title: name }))
      announceCreated()
      onDone()
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const chip = (on: boolean) =>
    `px-2 py-1.5 text-xs rounded border whitespace-nowrap ${on ? 'border-blue-500 text-blue-300' : 'border-zinc-700 text-zinc-300'} bg-zinc-800`
  const toggle = (p: Exclude<Pop, null>) => setPop((cur) => (cur === p ? null : p))
  const option = (on: boolean) =>
    `px-3 py-2 text-left text-sm rounded ${on ? 'bg-zinc-700 text-white' : 'text-zinc-300 hover:bg-zinc-800'}`

  return (
    <div data-testid="add-detailed-form" className="relative flex flex-col gap-2">
      <input
        ref={titleRef}
        data-testid="add-title"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        onKeyDown={(e) => { if (e.key === 'Enter') void submit() }}
        placeholder={t('addSheet.titlePlaceholder')}
        enterKeyHint="done"
        className="w-full bg-transparent text-white text-base focus:outline-none"
      />
      <input
        data-testid="add-description"
        value={desc}
        onChange={(e) => setDesc(e.target.value)}
        placeholder={t('addSheet.description')}
        className="w-full bg-transparent text-zinc-400 text-sm focus:outline-none"
      />

      {pop && (
        <div data-testid={`add-pop-${pop}`} className="absolute left-0 right-0 bottom-full mb-2 bg-zinc-900 border border-zinc-700 rounded-lg p-2 flex flex-col gap-1 shadow-xl max-h-72 overflow-auto">
          {pop === 'date' && (
            <>
              <button type="button" className={option(isToday)} onClick={() => { setDay(todayInZone(new Date(), zone)); setPop(null) }}>{t('addSheet.today')}</button>
              <button type="button" className={option(false)} onClick={() => { setDay(todayInZone(new Date(Date.now() + 86_400_000), zone)); setPop(null) }}>{t('addSheet.tomorrow')}</button>
              <input type="date" data-testid="add-date-input" value={day} onChange={(e) => { if (e.target.value) { setDay(e.target.value); setPop(null) } }} className="px-3 py-2 bg-black border border-zinc-700 rounded text-white text-sm" />
            </>
          )}
          {pop === 'time' && (
            <>
              <button type="button" className={option(time === null)} onClick={() => { manual.current.time = true; setTime(null); setPop(null) }}>{t('addSheet.noTime')}</button>
              <input
                type="time"
                data-testid="add-time-input"
                value={time === null ? '' : hhmm(time)}
                onChange={(e) => { const m = /^(\d{2}):(\d{2})$/.exec(e.target.value); if (m) { manual.current.time = true; setTime(Number(m[1]) * 60 + Number(m[2])) } }}
                className="px-3 py-2 bg-black border border-zinc-700 rounded text-white text-sm"
              />
              <div className="text-xs text-zinc-500 px-1 pt-1">{t('addSheet.duration')}</div>
              <div className="grid grid-cols-4 gap-1">
                {DURATIONS.map((m) => (
                  <button key={m} type="button" className={option(dur === m)} onClick={() => { manual.current.time = true; setDur(m) }}>{m < 60 ? `${m}′` : `${m / 60}h`.replace('1.5h', '1½h')}</button>
                ))}
              </div>
            </>
          )}
          {pop === 'prio' && (
            <>
              <button type="button" className={option(priority === null)} onClick={() => { setPriority(null); setPop(null) }}>{t('addSheet.noPriority')}</button>
              {PRIORITIES.map((p) => (
                <button key={p} type="button" data-testid={`add-prio-${p}`} className={option(priority === p)} onClick={() => { setPriority(p); setPop(null) }}>
                  ⚑ {tt(`priority.${p}`)}
                </button>
              ))}
            </>
          )}
          {pop === 'cal' && <CalendarPicker id="add-sheet-calendar" value={calendarId} onChange={(id) => { manual.current.cal = true; setCalendarId(id); setPop(null) }} />}
          {pop === 'more' && (
            <>
              <button type="button" data-testid="add-make-event" className={option(kind === 'event')} onClick={() => { setKindOverride('event'); setPop(null) }}>📅 {t('addSheet.makeEvent')}</button>
              <button type="button" data-testid="add-make-task" className={option(kind === 'task')} onClick={() => { setKindOverride('task'); setPop(null) }}>✓ {t('addSheet.makeTask')}</button>
            </>
          )}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-1.5">
        <button type="button" data-testid="add-chip-date" data-day={day} className={chip(true)} onClick={() => toggle('date')}>📅 {dayLabel}{isToday ? ` · ${t('addSheet.todayShort')}` : ''}</button>
        <button type="button" data-testid="add-chip-time" className={chip(time !== null)} onClick={() => toggle('time')}>
          ⏰ {time === null ? t('addSheet.time') : `${hhmm(time)} · ${dur}′`}
        </button>
        {kind === 'task' && (
          <button type="button" data-testid="add-chip-prio" className={chip(priority !== null)} onClick={() => toggle('prio')}>⚑{priority !== null ? ` ${priority}` : ''}</button>
        )}
        <button type="button" data-testid="add-chip-cal" className={chip(calendarId !== '')} onClick={() => toggle('cal')}>◉</button>
        <button type="button" data-testid="add-chip-more" className={chip(kindOverride !== null)} onClick={() => toggle('more')}>⋯</button>
        <button
          type="button"
          data-testid="add-create"
          disabled={busy || !title.trim()}
          onClick={() => void submit()}
          className="ml-auto px-3 py-1.5 text-sm rounded bg-blue-600 text-onaccent disabled:bg-zinc-700 disabled:text-zinc-500"
        >
          ✓
        </button>
      </div>
      <div data-testid="add-kind" data-kind={kind} className="text-xs text-zinc-500">
        {kind === 'event' ? `📅 ${t('addSheet.event')}` : `✓ ${t('addSheet.task')}`}
      </div>
    </div>
  )
}
