import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Loader2, Calendar, CheckCircle, XCircle, Clock, BookOpen } from 'lucide-react'
import { getReflections, type Reflection } from '../../api/reflections'

// ─── Helpers ────────────────────────────────────────────────────────────────

/** Returns the ISO week number (1-53) and year for a date */
function isoWeekKey(dateStr: string): string {
  const d = new Date(dateStr)
  // Thursday of the current week determines the ISO year
  const thursday = new Date(d)
  thursday.setDate(d.getDate() - ((d.getDay() + 6) % 7) + 3)
  const firstThursday = new Date(thursday.getFullYear(), 0, 4)
  const weekNum =
    1 +
    Math.round(
      ((thursday.getTime() - firstThursday.getTime()) / 86400000 -
        3 +
        ((firstThursday.getDay() + 6) % 7)) /
        7,
    )
  return `${thursday.getFullYear()}-W${String(weekNum).padStart(2, '0')}`
}

/** Returns the Monday of the ISO week for display */
function weekStart(weekKey: string): Date {
  const [yearStr, weekStr] = weekKey.split('-W')
  const year = parseInt(yearStr, 10)
  const week = parseInt(weekStr, 10)
  // Jan 4 is always in week 1
  const jan4 = new Date(year, 0, 4)
  const monday = new Date(jan4)
  monday.setDate(jan4.getDate() - ((jan4.getDay() + 6) % 7) + (week - 1) * 7)
  return monday
}

/** Average of non-null values, returns null if none */
function average(values: (number | null)[]): number | null {
  const nums = values.filter((v): v is number => v !== null)
  if (nums.length === 0) return null
  return Math.round((nums.reduce((a, b) => a + b, 0) / nums.length) * 10) / 10
}

// ─── Score Bar ───────────────────────────────────────────────────────────────

interface ScoreBarProps {
  label: string
  value: number | null
  color: string
}

function ScoreBar({ label, value, color }: ScoreBarProps) {
  if (value === null) return null
  const pct = Math.round((value / 10) * 100)
  return (
    <div className="flex items-center gap-2">
      <span className="w-16 text-xs text-zinc-400 shrink-0">{label}</span>
      <div className="flex-1 h-1.5 bg-zinc-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${color}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="w-6 text-right text-xs font-mono text-zinc-300 shrink-0">{value}</span>
    </div>
  )
}

// ─── Reflection Card ─────────────────────────────────────────────────────────

interface ReflectionCardProps {
  reflection: Reflection
  t: (key: string) => string
}

function ReflectionCard({ reflection, t }: ReflectionCardProps) {
  const date = new Date(reflection.createdAt)
  const hasScores = reflection.focus !== null || reflection.energy !== null || reflection.mood !== null

  return (
    <article className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 space-y-3">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex items-center gap-2 text-zinc-400 min-w-0">
          <Calendar className="w-4 h-4 shrink-0" />
          <time
            dateTime={reflection.createdAt}
            className="text-sm font-mono truncate"
          >
            {date.toLocaleDateString(undefined, {
              weekday: 'short',
              month: 'short',
              day: 'numeric',
              year: 'numeric',
            })}
          </time>
        </div>

        {/* Completion badges */}
        <div className="flex flex-wrap items-center gap-2 shrink-0">
          <span
            className={`flex items-center gap-1 text-xs px-2 py-0.5 rounded-full ${
              reflection.wasCompleted
                ? 'bg-green-900/40 text-green-400'
                : 'bg-red-900/30 text-red-400'
            }`}
          >
            {reflection.wasCompleted ? (
              <CheckCircle className="w-3 h-3" />
            ) : (
              <XCircle className="w-3 h-3" />
            )}
            {reflection.wasCompleted ? t('completed') : t('notCompleted')}
          </span>
          <span
            className={`flex items-center gap-1 text-xs px-2 py-0.5 rounded-full ${
              reflection.wasOnTime
                ? 'bg-blue-900/40 text-blue-400'
                : 'bg-yellow-900/30 text-yellow-400'
            }`}
          >
            <Clock className="w-3 h-3" />
            {reflection.wasOnTime ? t('onTime') : t('notOnTime')}
          </span>
        </div>
      </div>

      {/* Score bars */}
      {hasScores && (
        <div className="space-y-1.5">
          <ScoreBar label={t('focus')} value={reflection.focus} color="bg-purple-500" />
          <ScoreBar label={t('energy')} value={reflection.energy} color="bg-yellow-500" />
          <ScoreBar label={t('mood')} value={reflection.mood} color="bg-green-500" />
        </div>
      )}

      {/* Notes */}
      {reflection.notes && (
        <p className="text-sm text-zinc-300 border-t border-zinc-800 pt-2 leading-relaxed">
          {reflection.notes}
        </p>
      )}
    </article>
  )
}

// ─── Weekly Summary ───────────────────────────────────────────────────────────

interface WeeklySummaryProps {
  weekKey: string
  reflections: Reflection[]
  t: (key: string, opts?: Record<string, unknown>) => string
}

function WeeklySummary({ weekKey, reflections, t }: WeeklySummaryProps) {
  const start = weekStart(weekKey)
  const avgFocus = average(reflections.map((r) => r.focus))
  const avgEnergy = average(reflections.map((r) => r.energy))
  const avgMood = average(reflections.map((r) => r.mood))

  const hasAverages = avgFocus !== null || avgEnergy !== null || avgMood !== null

  return (
    <div className="bg-zinc-900/60 border border-zinc-800/60 rounded-lg px-4 py-3 space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-mono font-medium text-zinc-200">
          {t('weekOf', {
            date: start.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
          })}
        </h3>
        <span className="text-xs text-zinc-500">
          {t('reflectionCount', { count: reflections.length })}
        </span>
      </div>
      {hasAverages && (
        <div className="flex flex-wrap items-center gap-4 text-xs">
          {avgFocus !== null && (
            <span className="text-purple-400 font-mono">
              {t('avgFocus')}: <strong>{avgFocus}</strong>
            </span>
          )}
          {avgEnergy !== null && (
            <span className="text-yellow-400 font-mono">
              {t('avgEnergy')}: <strong>{avgEnergy}</strong>
            </span>
          )}
          {avgMood !== null && (
            <span className="text-green-400 font-mono">
              {t('avgMood')}: <strong>{avgMood}</strong>
            </span>
          )}
        </div>
      )}
    </div>
  )
}

// ─── Filter types ─────────────────────────────────────────────────────────────

type FilterRange = 'all' | 'week' | 'month'

function isWithinRange(createdAt: string, range: FilterRange): boolean {
  if (range === 'all') return true
  const now = new Date()
  const d = new Date(createdAt)
  if (range === 'week') {
    const weekAgo = new Date(now)
    weekAgo.setDate(now.getDate() - 7)
    return d >= weekAgo
  }
  if (range === 'month') {
    const monthAgo = new Date(now)
    monthAgo.setMonth(now.getMonth() - 1)
    return d >= monthAgo
  }
  return true
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function Reflections() {
  const { t } = useTranslation('reflections')
  const { t: tc } = useTranslation('common')

  const [reflections, setReflections] = useState<Reflection[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filterRange, setFilterRange] = useState<FilterRange>('all')

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await getReflections()
        setReflections(data)
      } catch (err) {
        console.error('Failed to load reflections:', err)
        setError(t('error.load'))
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [t])

  const filtered = useMemo(
    () => reflections.filter((r) => isWithinRange(r.createdAt, filterRange)),
    [reflections, filterRange],
  )

  // Group by ISO week, sorted newest first
  const byWeek = useMemo(() => {
    const map = new Map<string, Reflection[]>()
    for (const r of filtered) {
      const key = isoWeekKey(r.createdAt)
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(r)
    }
    // Sort week keys descending
    return Array.from(map.entries()).sort(([a], [b]) => b.localeCompare(a))
  }, [filtered])

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <header data-hint="reflections.list" className="px-4 sm:px-6 py-4 border-b border-zinc-800 bg-zinc-900">
        <div className="flex items-center justify-between mb-3">
          <div>
            <h1 className="text-xl font-mono font-semibold text-white">{t('title')}</h1>
            <p className="text-sm text-zinc-500 mt-0.5">{t('subtitle')}</p>
          </div>
        </div>

        {/* Filter controls */}
        <div className="flex flex-wrap items-center gap-2">
          {(['all', 'week', 'month'] as FilterRange[]).map((range) => (
            <button
              key={range}
              onClick={() => setFilterRange(range)}
              className={`px-3 py-1.5 text-sm font-mono rounded-lg transition-colors ${
                filterRange === range
                  ? 'bg-blue-600 text-white'
                  : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-white'
              }`}
            >
              {t(`filter.${range}`)}
            </button>
          ))}
        </div>
      </header>

      {/* Body */}
      <main className="flex-1 overflow-y-auto p-4 sm:p-6">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="w-8 h-8 text-zinc-400 animate-spin" />
            <span className="ml-3 text-zinc-400 font-mono">{tc('status.loading')}</span>
          </div>
        ) : error ? (
          <div className="flex items-center justify-center py-16">
            <p className="text-red-400 font-mono">{error}</p>
          </div>
        ) : filtered.length === 0 ? (
          /* Empty state */
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <BookOpen className="w-16 h-16 text-zinc-700 mb-4" />
            <h2 className="text-lg font-mono font-medium text-zinc-300 mb-2">
              {t('noReflections')}
            </h2>
            <p className="text-sm text-zinc-500 max-w-sm mb-6">{t('noReflectionsHint')}</p>
            <Link
              to="/calendar"
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-mono rounded-lg transition-colors"
            >
              <Calendar className="w-4 h-4" />
              {t('goToCalendar')}
            </Link>
          </div>
        ) : (
          <div className="space-y-8 max-w-2xl mx-auto">
            {byWeek.map(([weekKey, weekReflections]) => (
              <section key={weekKey}>
                {/* Weekly summary */}
                <div className="mb-3">
                  <WeeklySummary weekKey={weekKey} reflections={weekReflections} t={t} />
                </div>

                {/* Individual reflection cards */}
                <div className="space-y-3">
                  {weekReflections.map((reflection) => (
                    <ReflectionCard key={reflection.id} reflection={reflection} t={t} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
