import {
  createContext,
  useContext,
  useState,
  useEffect,
  useRef,
  useCallback,
  type ReactNode,
} from 'react'
import {
  type TimerMode,
  type PomodoroSettings,
  type PersistedTimerState,
  type Completion,
  type SessionRecord,
} from '../lib/pomodoro/types'
import {
  loadSettings,
  saveSettings,
  loadTimerState,
  saveTimerState,
  isStale,
} from '../lib/pomodoro/storage'
import { nextPhase, durationMsForMode, minutesForMode } from '../lib/pomodoro/machine'
import { recordWorkCompletion, undoWorkCompletion } from '../lib/pomodoro/tracking'
import { playBeep, requestNotificationPermission, showNotification } from '../lib/pomodoro/notify'

interface PomodoroContextValue {
  phase: TimerMode
  remainingMs: number
  isRunning: boolean
  isActive: boolean
  sessionsCompleted: number
  linkedTaskId: string | null
  linkedTaskTitle: string | null
  settings: PomodoroSettings
  completions: Array<Completion & { id: number }>
  breakOver: boolean
  interruptedWhileAway: boolean
  start: () => void
  pause: () => void
  reset: () => void
  skip: () => void
  selectPhase: (m: TimerMode) => void
  setLinkedTask: (id: string | null, title: string | null) => void
  updateSettings: (patch: Partial<PomodoroSettings>) => void
  dismissCompletion: (id: number) => void
  undoCompletion: (id: number) => void
  dismissBreakOver: () => void
  dismissInterrupted: () => void
}

const PomodoroContext = createContext<PomodoroContextValue | null>(null)

const HISTORY_KEY = 'nb-pomodoro-history'

function appendHistory(record: SessionRecord): void {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    const list: SessionRecord[] = raw ? JSON.parse(raw) : []
    list.push(record)
    localStorage.setItem(HISTORY_KEY, JSON.stringify(list))
  } catch {
    // ignore persistence failure
  }
}

/**
 * Computes the corrected initial state ONCE, synchronously, before first render.
 * A stale persisted block (endsAt already in the past — elapsed while the app
 * was closed) is resolved to idle here so it is NEVER committed as isRunning:true.
 * This is critical: a mount EFFECT cannot do this safely, because the completion
 * effect runs in the same passive-effect flush and would still see isRunning:true
 * and fire completePhase() — fabricating a calendar event/time that decision (d)
 * of the spec forbids.
 */
function computeInitial(
  p: PersistedTimerState | null,
  s: PomodoroSettings
): PersistedTimerState & { interrupted: boolean } {
  if (!p) {
    return {
      phase: 'work',
      endsAt: null,
      isRunning: false,
      remainingWhenPaused: durationMsForMode('work', s),
      sessionsCompleted: 0,
      linkedTaskId: null,
      linkedTaskTitle: null,
      blockStartedAt: null,
      interrupted: false,
    }
  }
  if (isStale(p, Date.now())) {
    return {
      ...p,
      isRunning: false,
      endsAt: null,
      remainingWhenPaused: durationMsForMode(p.phase, s),
      blockStartedAt: null,
      interrupted: true,
    }
  }
  return { ...p, interrupted: false }
}

export function PomodoroProvider({ children }: { children: ReactNode }) {
  const initialSettings = useRef(loadSettings()).current
  const init = useRef(computeInitial(loadTimerState(), initialSettings)).current

  const [settings, setSettings] = useState<PomodoroSettings>(initialSettings)
  const [phase, setPhase] = useState<TimerMode>(init.phase)
  const [endsAt, setEndsAt] = useState<number | null>(init.endsAt)
  const [isRunning, setIsRunning] = useState<boolean>(init.isRunning)
  const [remainingWhenPaused, setRemainingWhenPaused] = useState<number | null>(init.remainingWhenPaused)
  const [sessionsCompleted, setSessionsCompleted] = useState<number>(init.sessionsCompleted)
  const [linkedTaskId, setLinkedTaskId] = useState<string | null>(init.linkedTaskId)
  const [linkedTaskTitle, setLinkedTaskTitle] = useState<string | null>(init.linkedTaskTitle)
  const [blockStartedAt, setBlockStartedAt] = useState<string | null>(init.blockStartedAt)

  const [now, setNow] = useState<number>(() => Date.now())
  const [completions, setCompletions] = useState<Array<Completion & { id: number }>>([])
  const completionIdRef = useRef(0)
  const completionsRef = useRef<Array<Completion & { id: number }>>([])
  useEffect(() => { completionsRef.current = completions }, [completions])
  const [breakOver, setBreakOver] = useState(false)
  const [interruptedWhileAway, setInterruptedWhileAway] = useState(init.interrupted)

  const handledEndsAtRef = useRef<number | null>(null)

  const remainingMs = isRunning
    ? Math.max(0, (endsAt ?? now) - now)
    : remainingWhenPaused ?? durationMsForMode(phase, settings)

  const fullMsForPhase = durationMsForMode(phase, settings)
  const isActive = isRunning || (remainingWhenPaused != null && remainingWhenPaused < fullMsForPhase)

  // (Stale-block resolution happens synchronously in computeInitial above —
  // never in an effect, to avoid the completion effect firing on a stale block.)

  // Display tick — re-render once per second while running. Display-only;
  // remainingMs is always derived from the wall clock, not this interval.
  useEffect(() => {
    if (!isRunning) return
    setNow(Date.now())
    const id = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(id)
  }, [isRunning])

  // Persist whenever any durable field changes.
  useEffect(() => {
    saveTimerState({
      phase,
      endsAt,
      isRunning,
      remainingWhenPaused,
      sessionsCompleted,
      linkedTaskId,
      linkedTaskTitle,
      blockStartedAt,
    })
  }, [phase, endsAt, isRunning, remainingWhenPaused, sessionsCompleted, linkedTaskId, linkedTaskTitle, blockStartedAt])

  // Persist settings whenever they change (runs once on mount, then on every update).
  useEffect(() => { saveSettings(settings) }, [settings])

  const seedPhase = useCallback(
    (mode: TimerMode, autoStart: boolean) => {
      const durMs = durationMsForMode(mode, settings)
      setPhase(mode)
      if (autoStart) {
        setEndsAt(Date.now() + durMs)
        setRemainingWhenPaused(null)
        setIsRunning(true)
        setBlockStartedAt(new Date().toISOString())
      } else {
        setEndsAt(null)
        setRemainingWhenPaused(durMs)
        setIsRunning(false)
        setBlockStartedAt(null)
      }
    },
    [settings]
  )

  const completePhase = useCallback(() => {
    const finishedPhase = phase
    const finishedEndsAt = endsAt
    const endedISO = finishedEndsAt != null ? new Date(finishedEndsAt).toISOString() : new Date().toISOString()

    if (finishedPhase === 'work') {
      const minutes = minutesForMode('work', settings)
      appendHistory({
        date: endedISO.slice(0, 10),
        mode: 'work',
        durationSeconds: minutes * 60,
        taskId: linkedTaskId ?? undefined,
        completedAt: endedISO,
      })
      showNotification('Work session complete', 'Time for a break')
      const startedISO = blockStartedAt ?? new Date(Date.now() - minutes * 60 * 1000).toISOString()
      void recordWorkCompletion({
        taskId: linkedTaskId,
        taskTitle: linkedTaskTitle,
        startedAtISO: startedISO,
        endsAtISO: endedISO,
        minutes,
      }).then((res) => {
        const id = completionIdRef.current++
        setCompletions((prev) => [...prev, { ...res, id }])
      })

      const newCompleted = sessionsCompleted + 1
      setSessionsCompleted(newCompleted)
      seedPhase(nextPhase('work', newCompleted, settings.sessionsBeforeLong), settings.autoStartBreaks)
    } else {
      appendHistory({
        date: endedISO.slice(0, 10),
        mode: finishedPhase,
        durationSeconds: minutesForMode(finishedPhase, settings) * 60,
        completedAt: endedISO,
      })
      showNotification('Break over', 'Back to focus?')
      setBreakOver(true)
      seedPhase('work', false)
    }
  }, [phase, endsAt, settings, linkedTaskId, linkedTaskTitle, blockStartedAt, sessionsCompleted, seedPhase])

  // Completion detector — fires exactly once per block. Never inside a setState updater.
  useEffect(() => {
    if (!isRunning || endsAt == null) return
    if (now < endsAt) return
    if (handledEndsAtRef.current === endsAt) return
    handledEndsAtRef.current = endsAt
    playBeep(settings.soundEnabled)
    completePhase()
  }, [now, isRunning, endsAt, settings.soundEnabled, completePhase])

  const start = useCallback(() => {
    requestNotificationPermission()
    const durMs = remainingWhenPaused ?? durationMsForMode(phase, settings)
    setEndsAt(Date.now() + durMs)
    setRemainingWhenPaused(null)
    setIsRunning(true)
    setBlockStartedAt((prev) => prev ?? new Date().toISOString())
    setInterruptedWhileAway(false)
  }, [remainingWhenPaused, phase, settings])

  const pause = useCallback(() => {
    setRemainingWhenPaused(endsAt != null ? Math.max(0, endsAt - Date.now()) : remainingWhenPaused)
    setEndsAt(null)
    setIsRunning(false)
  }, [endsAt, remainingWhenPaused])

  const reset = useCallback(() => {
    setIsRunning(false)
    setEndsAt(null)
    setRemainingWhenPaused(durationMsForMode(phase, settings))
    setBlockStartedAt(null)
  }, [phase, settings])

  const skip = useCallback(() => {
    // Advance phase only. No tracking, no increment — you didn't complete this block.
    const np: TimerMode = phase === 'work' ? 'shortBreak' : 'work'
    seedPhase(np, false)
  }, [phase, seedPhase])

  const selectPhase = useCallback(
    (m: TimerMode) => {
      seedPhase(m, false)
    },
    [seedPhase]
  )

  const setLinkedTask = useCallback((id: string | null, title: string | null) => {
    setLinkedTaskId(id)
    setLinkedTaskTitle(title)
  }, [])

  const updateSettings = useCallback((patch: Partial<PomodoroSettings>) => {
    setSettings((prev) => ({ ...prev, ...patch }))
  }, [])

  const dismissCompletion = useCallback((id: number) => {
    setCompletions((prev) => prev.filter((c) => c.id !== id))
  }, [])
  const undoCompletion = useCallback((id: number) => {
    const c = completionsRef.current.find((x) => x.id === id)
    if (c) void undoWorkCompletion(c)
    setCompletions((prev) => prev.filter((x) => x.id !== id))
  }, [])
  const dismissBreakOver = useCallback(() => setBreakOver(false), [])
  const dismissInterrupted = useCallback(() => setInterruptedWhileAway(false), [])

  const value: PomodoroContextValue = {
    phase,
    remainingMs,
    isRunning,
    isActive,
    sessionsCompleted,
    linkedTaskId,
    linkedTaskTitle,
    settings,
    completions,
    breakOver,
    interruptedWhileAway,
    start,
    pause,
    reset,
    skip,
    selectPhase,
    setLinkedTask,
    updateSettings,
    dismissCompletion,
    undoCompletion,
    dismissBreakOver,
    dismissInterrupted,
  }

  return <PomodoroContext.Provider value={value}>{children}</PomodoroContext.Provider>
}

export function usePomodoro(): PomodoroContextValue {
  const ctx = useContext(PomodoroContext)
  if (!ctx) throw new Error('usePomodoro must be used within PomodoroProvider')
  return ctx
}
