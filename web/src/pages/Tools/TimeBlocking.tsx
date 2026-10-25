import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Clock, Plus, Trash2, AlertTriangle } from 'lucide-react'
import { useAuthContext } from '../../contexts/AuthContext'
import { readBudgetCategories, workDayCount, workHoursPerDay, type TimeCategory } from '../../lib/tools/timeBudget'

// ─── Types ────────────────────────────────────────────────────────────────────

interface TimeBlockingState {
  categories: TimeCategory[]
}

// ─── Default categories ───────────────────────────────────────────────────────

const DEFAULT_CATEGORIES: TimeCategory[] = [
  {
    id: 'deep-work',
    labelKey: 'timeBlocking.cat.deepWork',
    customLabel: '',
    hours: 4,
    color: 'bg-blue-500',
    isCustom: false,
  },
  {
    id: 'meetings',
    labelKey: 'timeBlocking.cat.meetings',
    customLabel: '',
    hours: 1.5,
    color: 'bg-purple-500',
    isCustom: false,
  },
  {
    id: 'admin',
    labelKey: 'timeBlocking.cat.admin',
    customLabel: '',
    hours: 0.5,
    color: 'bg-yellow-500',
    isCustom: false,
  },
  {
    id: 'breaks',
    labelKey: 'timeBlocking.cat.breaks',
    customLabel: '',
    hours: 1,
    color: 'bg-green-500',
    isCustom: false,
  },
  {
    id: 'buffer',
    labelKey: 'timeBlocking.cat.buffer',
    customLabel: '',
    hours: 1,
    color: 'bg-orange-500',
    isCustom: false,
  },
]

const CUSTOM_COLORS = [
  'bg-pink-500',
  'bg-teal-500',
  'bg-indigo-500',
  'bg-red-500',
  'bg-cyan-500',
  'bg-lime-500',
]

// ─── Where the split comes from ───────────────────────────────────────────────

// Until 25.09 the whole tool lived in this key of one browser. It is read once,
// only when the account has no split yet, so nothing typed there is lost.
const LEGACY_KEY = 'nb-time-blocking'

function legacyCategories(): TimeCategory[] | null {
  try {
    const raw = localStorage.getItem(LEGACY_KEY)
    return raw ? readBudgetCategories((JSON.parse(raw) as { categories?: unknown }).categories) : null
  } catch {
    return null
  }
}

// ─── Stacked bar ──────────────────────────────────────────────────────────────

interface StackedBarProps {
  categories: TimeCategory[]
  totalAvailable: number
  labelFn: (cat: TimeCategory) => string
}

function StackedBar({ categories, totalAvailable, labelFn }: StackedBarProps) {
  const { t } = useTranslation('tools')
  const totalAllocated = categories.reduce((s, c) => s + c.hours, 0)
  const overBudget = totalAllocated > totalAvailable

  return (
    <div className="space-y-2">
      {/* Bar */}
      <div className="w-full h-8 rounded-lg overflow-hidden bg-zinc-800 border border-zinc-700 flex">
        {categories
          .filter((c) => c.hours > 0)
          .map((cat) => {
            const pct = Math.min((cat.hours / Math.max(totalAllocated, totalAvailable)) * 100, 100)
            return (
              <div
                key={cat.id}
                className={`${cat.color} transition-all duration-300 flex items-center justify-center overflow-hidden`}
                style={{ width: `${pct}%` }}
                title={`${labelFn(cat)}: ${cat.hours}h`}
              >
                {pct > 8 && (
                  <span className="text-[10px] font-medium text-white/90 truncate px-1">
                    {cat.hours}h
                  </span>
                )}
              </div>
            )
          })}
        {/* Remaining */}
        {!overBudget && totalAllocated < totalAvailable && (
          <div
            className="bg-zinc-700/50 flex-1 flex items-center justify-center"
            title={t('timeBlocking.remaining')}
          >
            <span className="text-[10px] text-zinc-500">
              {(totalAvailable - totalAllocated).toFixed(1)}h
            </span>
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-x-3 gap-y-1">
        {categories
          .filter((c) => c.hours > 0)
          .map((cat) => (
            <div key={cat.id} className="flex items-center gap-1.5">
              <span className={`w-2 h-2 rounded-sm flex-shrink-0 ${cat.color}`} />
              <span className="text-xs text-zinc-400">{labelFn(cat)}</span>
            </div>
          ))}
      </div>
    </div>
  )
}

// ─── Summary card ─────────────────────────────────────────────────────────────

interface SummaryProps {
  totalAvailable: number
  totalAllocated: number
  utilization: number
}

function Summary({ totalAvailable, totalAllocated, utilization }: SummaryProps) {
  const { t } = useTranslation('tools')
  const overBudget = utilization > 100
  const remaining = totalAvailable - totalAllocated

  return (
    <div className="grid grid-cols-3 gap-3">
      <div className="bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2.5 text-center">
        <p className="text-xs text-zinc-500 mb-0.5">{t('timeBlocking.totalAllocated')}</p>
        <p className="text-lg font-bold font-mono text-zinc-100">{totalAllocated.toFixed(1)}h</p>
      </div>
      <div
        className={`border rounded-lg px-3 py-2.5 text-center transition-colors ${
          remaining < 0
            ? 'bg-red-900/20 border-red-800'
            : remaining < 1
            ? 'bg-yellow-900/20 border-yellow-800'
            : 'bg-zinc-800 border-zinc-700'
        }`}
      >
        <p className="text-xs text-zinc-500 mb-0.5">{t('timeBlocking.remaining')}</p>
        <p
          className={`text-lg font-bold font-mono ${
            remaining < 0 ? 'text-red-400' : remaining < 1 ? 'text-yellow-400' : 'text-green-400'
          }`}
        >
          {remaining.toFixed(1)}h
        </p>
      </div>
      <div
        className={`border rounded-lg px-3 py-2.5 text-center transition-colors ${
          overBudget ? 'bg-red-900/20 border-red-800' : 'bg-zinc-800 border-zinc-700'
        }`}
      >
        <p className="text-xs text-zinc-500 mb-0.5">{t('timeBlocking.utilization')}</p>
        <p
          className={`text-lg font-bold font-mono ${overBudget ? 'text-red-400' : 'text-zinc-100'}`}
        >
          {utilization.toFixed(0)}%
        </p>
      </div>
    </div>
  )
}

// ─── Main Page ────────────────────────────────────────────────────────────────

// ─── Category row ─────────────────────────────────────────────────────────────

interface CategoryRowProps {
  cat: TimeCategory
  label: string
  onHoursChange: (id: string, hours: number) => void
  onRemove: (id: string) => void
}

function CategoryRow({ cat, label, onHoursChange, onRemove }: CategoryRowProps) {
  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const val = parseFloat(e.target.value)
    if (!Number.isNaN(val) && val >= 0) {
      onHoursChange(cat.id, Math.min(val, 24))
    }
  }

  return (
    <div className="flex items-center gap-3 py-2.5 border-b border-zinc-800 last:border-0">
      {/* Color swatch */}
      <span className={`w-3 h-3 rounded-sm flex-shrink-0 ${cat.color}`} />

      {/* Label */}
      <span className="flex-1 text-sm text-zinc-300 truncate">{label}</span>

      {/* Hours input */}
      <div className="flex items-center gap-1.5">
        <input
          type="number"
          min="0"
          max="24"
          step="0.5"
          value={cat.hours}
          onChange={handleChange}
          className="w-16 bg-zinc-800 border border-zinc-700 rounded-md px-2 py-1 text-sm font-mono
            text-zinc-100 text-right focus:outline-none focus:border-blue-500 transition-colors"
        />
        <span className="text-xs text-zinc-500">h</span>
      </div>

      {/* Remove (custom only) */}
      {cat.isCustom && (
        <button
          onClick={() => onRemove(cat.id)}
          className="text-zinc-600 hover:text-red-400 transition-colors"
          title="Remove"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  )
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function TimeBlocking() {
  const { t } = useTranslation('tools')

  const { user, updateSettings } = useAuthContext()
  const [state, setState] = useState<TimeBlockingState>(() => ({
    categories: readBudgetCategories(user?.settings?.time_budget) ?? legacyCategories() ?? DEFAULT_CATEGORIES,
  }))
  const [newCategoryLabel, setNewCategoryLabel] = useState('')
  const [showAddForm, setShowAddForm] = useState(false)

  // The day's length is the account's work hours (⚙️), not a number typed here.
  const workHours = workHoursPerDay(user?.settings?.work_start, user?.settings?.work_end)
  const hoursPerDay = workHours ?? 8
  const workDays = workDayCount(user?.settings?.work_days)

  // Saved to the account, a moment after the last change (typing hours would
  // otherwise send a request per keystroke). The first render saves nothing.
  const firstRender = useRef(true)
  useEffect(() => {
    if (firstRender.current) {
      firstRender.current = false
      return
    }
    const timer = window.setTimeout(() => {
      void updateSettings({ time_budget: state.categories }).catch(() => undefined)
    }, 700)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- saves on the split changing only
  }, [state.categories])

  const labelFn = useCallback(
    (cat: TimeCategory) => (cat.isCustom ? cat.customLabel : t(cat.labelKey)),
    [t]
  )

  const totalAvailable = hoursPerDay
  const totalAllocated = state.categories.reduce((s, c) => s + c.hours, 0)
  const utilization = totalAvailable > 0 ? (totalAllocated / totalAvailable) * 100 : 0
  const overBudget = utilization > 100

  // ── Handlers ─────────────────────────────────────────────────────────────────

  function handleHoursChange(id: string, hours: number) {
    setState((prev) => ({
      ...prev,
      categories: prev.categories.map((c) => (c.id === id ? { ...c, hours } : c)),
    }))
  }

  function handleRemoveCategory(id: string) {
    setState((prev) => ({
      ...prev,
      categories: prev.categories.filter((c) => c.id !== id),
    }))
  }

  function handleAddCategory() {
    const label = newCategoryLabel.trim()
    if (!label) return
    const colorIndex = state.categories.filter((c) => c.isCustom).length % CUSTOM_COLORS.length
    const newCat: TimeCategory = {
      id: `custom-${Date.now()}`,
      labelKey: '',
      customLabel: label,
      hours: 0,
      color: CUSTOM_COLORS[colorIndex],
      isCustom: true,
    }
    setState((prev) => ({ ...prev, categories: [...prev.categories, newCat] }))
    setNewCategoryLabel('')
    setShowAddForm(false)
  }

  function handleAddKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') handleAddCategory()
    if (e.key === 'Escape') {
      setShowAddForm(false)
      setNewCategoryLabel('')
    }
  }

  // ── Render ──────────────────────────────────────────────────────────────────
  return (
    <div className="min-h-[calc(100vh-56px)] bg-zinc-950 p-4 sm:p-6">
      <div data-hint="tools.timeBlocking.grid" className="max-w-2xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex items-center gap-2.5">
          <Clock className="w-5 h-5 text-green-400" />
          <h1 className="text-xl font-bold text-zinc-100">{t('timeBlocking.title')}</h1>
        </div>

        {/* ── Available hours: the account's work hours ── */}
        <section data-testid="time-budget-hours" className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 space-y-2">
          <h2 className="text-sm font-semibold text-zinc-300">{t('timeBlocking.availableHours')}</h2>
          <p className="text-sm font-mono text-zinc-100">
            {workHours === null
              ? t('timeBlocking.noWorkHours')
              : t('timeBlocking.fromWorkHours', {
                  hours: hoursPerDay,
                  start: user?.settings?.work_start,
                  end: user?.settings?.work_end,
                  days: workDays,
                  week: Math.round(hoursPerDay * workDays * 10) / 10,
                })}
          </p>
          <Link to="/settings" className="text-xs text-blue-400 hover:underline">
            {t('timeBlocking.changeWorkHours')}
          </Link>
        </section>

        {/* ── Time categories ── */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-zinc-300">{t('timeBlocking.categories')}</h2>
            <button
              onClick={() => setShowAddForm((v) => !v)}
              className="flex items-center gap-1 px-2 py-1 text-xs bg-zinc-800 border border-zinc-700
                text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700 rounded-lg transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />
              {t('timeBlocking.addCategory')}
            </button>
          </div>

          {/* Category list */}
          <div className="divide-y divide-zinc-800/0">
            {state.categories.map((cat) => (
              <CategoryRow
                key={cat.id}
                cat={cat}
                label={labelFn(cat)}
                onHoursChange={handleHoursChange}
                onRemove={handleRemoveCategory}
              />
            ))}
          </div>

          {/* Add form */}
          {showAddForm && (
            <div className="flex gap-2 pt-1">
              <input
                autoFocus
                type="text"
                value={newCategoryLabel}
                onChange={(e) => setNewCategoryLabel(e.target.value)}
                onKeyDown={handleAddKeyDown}
                placeholder={t('timeBlocking.categoryPlaceholder')}
                className="flex-1 bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-1.5 text-sm
                  text-zinc-100 placeholder-zinc-600 focus:outline-none focus:border-blue-500 transition-colors"
              />
              <button
                onClick={handleAddCategory}
                disabled={!newCategoryLabel.trim()}
                className="px-3 py-1.5 text-xs bg-blue-600 hover:bg-blue-500 disabled:opacity-40
                  disabled:cursor-not-allowed text-white rounded-lg transition-colors"
              >
                {t('timeBlocking.add')}
              </button>
            </div>
          )}
        </section>

        {/* ── Visual breakdown ── */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 space-y-4">
          <h2 className="text-sm font-semibold text-zinc-300">{t('timeBlocking.breakdown')}</h2>

          {/* Over budget warning */}
          {overBudget && (
            <div className="flex items-center gap-2 px-3 py-2 bg-red-900/20 border border-red-800
              rounded-lg text-sm text-red-400">
              <AlertTriangle className="w-4 h-4 flex-shrink-0" />
              {t('timeBlocking.overBudget', {
                over: (totalAllocated - totalAvailable).toFixed(1),
              })}
            </div>
          )}

          <StackedBar
            categories={state.categories}
            totalAvailable={totalAvailable}
            labelFn={labelFn}
          />
        </section>

        {/* ── Summary ── */}
        <Summary
          totalAvailable={totalAvailable}
          totalAllocated={totalAllocated}
          utilization={utilization}
        />

      </div>
    </div>
  )
}
