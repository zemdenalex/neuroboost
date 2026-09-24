import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarDays } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import type { UserSettings } from '../../../api/auth'
import { MONTH_VARIANTS, readMonthVariant, type MonthVariant } from '../../../lib/calendar/monthVariant'

interface Props {
  /** The page's debounced saver (see UIScaleSection for why it is passed in). */
  autoSave: (patch: Partial<UserSettings>) => void
}

/** A static sketch of one cell of each variant: enough to tell them apart. */
function Sketch({ variant }: { variant: MonthVariant }) {
  const cell = 'h-10 rounded-sm border border-zinc-700 bg-black p-0.5 flex flex-col gap-0.5 overflow-hidden'
  const cells = [0, 1, 2]
  switch (variant) {
    case 'list':
      return (
        <div className="grid grid-cols-3 gap-0.5">
          {cells.map((i) => (
            <div key={i} className={cell}>
              <div className="h-1 w-3/4 border-l-2 border-blue-500 bg-zinc-700" />
              <div className="h-1 w-2/3 border-l-2 border-green-500 bg-zinc-700" />
              {i === 1 && <div className="h-1 w-1/2 border-l-2 border-amber-500 bg-zinc-700" />}
            </div>
          ))}
        </div>
      )
    case 'classic':
      return (
        <div className="grid grid-cols-3 gap-0.5">
          {cells.map((i) => (
            <div key={i} className={`${cell} ${i === 0 ? 'bg-green-950' : ''}`}>
              <div className="h-1.5 w-full rounded-sm bg-blue-600" />
              <div className="h-1.5 w-full rounded-sm bg-green-600" />
            </div>
          ))}
        </div>
      )
    case 'heat':
      return (
        <div className="grid grid-cols-3 gap-0.5">
          {[70, 30, 90].map((h, i) => (
            <div key={i} className={`${cell} justify-end p-0`}>
              <div className={i === 1 ? 'bg-orange-500' : 'bg-green-500'} style={{ height: `${h}%` }} />
            </div>
          ))}
        </div>
      )
    case 'split':
      return (
        <div className="flex flex-col gap-0.5">
          <div className="grid grid-cols-3 gap-0.5">
            {cells.map((i) => (
              <div key={i} className="h-4 rounded-sm border border-zinc-700 bg-black flex items-end justify-center gap-px pb-0.5">
                <span className="w-1 h-1 rounded-full bg-blue-500" />
                {i !== 2 && <span className="w-1 h-1 rounded-full bg-green-500" />}
              </div>
            ))}
          </div>
          <div className="h-1 w-full bg-zinc-700" />
          <div className="h-1 w-2/3 bg-zinc-700" />
        </div>
      )
    case 'commit':
      return (
        <div className="grid grid-cols-3 gap-0.5">
          {[100, 60, 20].map((w, i) => (
            <div key={i} className={cell}>
              <div className="h-1 w-full bg-zinc-800">
                <div className={w === 100 ? 'h-full bg-green-500' : w === 60 ? 'h-full bg-orange-500' : 'h-full bg-red-600'} style={{ width: `${w}%` }} />
              </div>
              <div className="text-[6px] leading-none text-zinc-400">☑ ☑ ☐</div>
            </div>
          ))}
        </div>
      )
  }
}

/**
 * Which month view the calendar shows (spec V003-20260924-arc-web-month-view).
 * Denis, 24.09: «let's build all of them, make a default and other to choose in settings».
 */
export function MonthViewSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [variant, setVariant] = useState<MonthVariant>(() => readMonthVariant(user?.settings))

  // Keyed on the account, not the user object: see UIScaleSection.
  useEffect(() => {
    setVariant(readMonthVariant(user?.settings))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const choose = (next: MonthVariant) => {
    setVariant(next)
    autoSave({ month_view_variant: next })
  }

  return (
    <section data-testid="month-view-section" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-1">
        <CalendarDays className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('monthView.title')}</h2>
      </div>
      <p className="text-xs text-zinc-500 mb-4">{t('monthView.note')}</p>
      <div role="radiogroup" aria-label={t('monthView.title')} className="grid grid-cols-2 md:grid-cols-5 gap-3">
        {MONTH_VARIANTS.map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={variant === v}
            data-testid={`month-variant-${v}`}
            onClick={() => choose(v)}
            className={`text-left p-2 rounded-lg border transition-colors flex flex-col gap-2 ${
              variant === v ? 'bg-blue-600/20 border-blue-500' : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
            }`}
          >
            <Sketch variant={v} />
            <span className={`text-sm font-mono ${variant === v ? 'text-blue-300' : 'text-zinc-200'}`}>
              {t(`monthView.${v}.name`)}
            </span>
            <span className="text-xs text-zinc-500">{t(`monthView.${v}.desc`)}</span>
          </button>
        ))}
      </div>
    </section>
  )
}
