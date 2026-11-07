import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Hand } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import type { UserSettings } from '../../../api/auth'
import {
  PHONE_DRAG_DELAYS,
  PHONE_DRAG_STARTS,
  PHONE_DRAG_STEPS,
  PHONE_RESIZES,
  readPhoneDrag,
  type PhoneDragPrefs,
} from '../../../lib/calendar/phoneDrag'

interface Props {
  /** The page's debounced saver (see UIScaleSection for why it is passed in). */
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * ⚙️ «Перетаскивание на телефоне» — variant 4 of
 * docs/V003-20261025-var-perenos-na-telefone.html, step 10 of
 * docs/tasks-phone-drag.md. All six choices are top-level settings keys read by
 * lib/calendar/phoneDrag.ts readPhoneDrag; the calendar reads the same function,
 * so what is shown here is what the grid does. Shown on the computer too: the
 * choice is for the phone, made wherever the person happens to be.
 */
export function PhoneDragSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [prefs, setPrefs] = useState<PhoneDragPrefs>(() => readPhoneDrag(user?.settings))

  // Keyed on the account, not the user object: see UIScaleSection.
  useEffect(() => {
    setPrefs(readPhoneDrag(user?.settings))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const set = <K extends keyof PhoneDragPrefs>(key: K, value: PhoneDragPrefs[K], patch: Partial<UserSettings>) => {
    setPrefs((p) => ({ ...p, [key]: value }))
    autoSave(patch)
  }

  const chip = (on: boolean) =>
    `px-3 py-1.5 rounded-lg border text-sm font-mono transition-colors ${
      on ? 'bg-blue-600/20 border-blue-500 text-blue-300' : 'bg-zinc-800 border-zinc-700 text-zinc-200 hover:border-zinc-600'
    }`
  const card = (on: boolean) =>
    `text-left p-2 rounded-lg border transition-colors flex flex-col gap-1 min-w-0 ${
      on ? 'bg-blue-600/20 border-blue-500' : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
    }`

  return (
    <section data-testid="phone-drag-section" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-1">
        <Hand className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('phoneDrag.title')}</h2>
      </div>
      <p className="text-xs text-zinc-500 mb-3">{t('phoneDrag.note')}</p>

      <h3 className="text-sm font-mono font-semibold text-zinc-200 mb-2">{t('phoneDrag.start.label')}</h3>
      <div role="radiogroup" aria-label={t('phoneDrag.start.label')} className="grid grid-cols-3 gap-2 md:w-[30rem]">
        {PHONE_DRAG_STARTS.map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={prefs.start === v}
            data-testid={`phone-drag-start-${v}`}
            onClick={() => set('start', v, { phone_drag_start: v })}
            className={card(prefs.start === v)}
          >
            <span className={`text-sm font-mono ${prefs.start === v ? 'text-blue-300' : 'text-zinc-200'}`}>
              {t(`phoneDrag.start.${v}.name`)}
            </span>
            <span className="text-xs text-zinc-500">{t(`phoneDrag.start.${v}.desc`)}</span>
          </button>
        ))}
      </div>

      {prefs.start !== 'handle' && (
        <>
          <h3 className="mt-5 text-sm font-mono font-semibold text-zinc-200 mb-2">{t('phoneDrag.delay')}</h3>
          <div role="radiogroup" aria-label={t('phoneDrag.delay')} className="flex flex-wrap gap-2">
            {PHONE_DRAG_DELAYS.map((v) => (
              <button
                key={v}
                type="button"
                role="radio"
                aria-checked={prefs.delayMs === v}
                data-testid={`phone-drag-delay-${v}`}
                onClick={() => set('delayMs', v, { phone_drag_delay_ms: v })}
                className={chip(prefs.delayMs === v)}
              >
                {t('phoneDrag.ms', { n: v })}
              </button>
            ))}
          </div>
        </>
      )}

      <h3 className="mt-5 text-sm font-mono font-semibold text-zinc-200 mb-2">{t('phoneDrag.step')}</h3>
      <div role="radiogroup" aria-label={t('phoneDrag.step')} className="flex flex-wrap gap-2">
        {PHONE_DRAG_STEPS.map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={prefs.stepMin === v}
            data-testid={`phone-drag-step-${v}`}
            onClick={() => set('stepMin', v, { phone_drag_step_min: v })}
            className={chip(prefs.stepMin === v)}
          >
            {t('phoneDrag.min', { n: v })}
          </button>
        ))}
      </div>

      <h3 className="mt-5 text-sm font-mono font-semibold text-zinc-200 mb-2">{t('phoneDrag.resize.label')}</h3>
      <div role="radiogroup" aria-label={t('phoneDrag.resize.label')} className="grid grid-cols-3 gap-2 md:w-[30rem]">
        {PHONE_RESIZES.map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={prefs.resize === v}
            data-testid={`phone-drag-resize-${v}`}
            onClick={() => set('resize', v, { phone_drag_resize: v })}
            className={card(prefs.resize === v)}
          >
            <span className={`text-sm font-mono ${prefs.resize === v ? 'text-blue-300' : 'text-zinc-200'}`}>
              {t(`phoneDrag.resize.${v}.name`)}
            </span>
            <span className="text-xs text-zinc-500">{t(`phoneDrag.resize.${v}.desc`)}</span>
          </button>
        ))}
      </div>

      <div className="mt-5 flex flex-col gap-2 md:w-[30rem]">
        <Switch
          testId="phone-drag-edge-flip"
          on={prefs.edgeFlip}
          label={t('phoneDrag.edgeFlip.label')}
          hint={t('phoneDrag.edgeFlip.hint')}
          onClick={() => set('edgeFlip', !prefs.edgeFlip, { phone_drag_edge_flip: !prefs.edgeFlip })}
        />
        <Switch
          testId="phone-drag-vibrate"
          on={prefs.vibrate}
          label={t('phoneDrag.vibrate.label')}
          hint={t('phoneDrag.vibrate.hint')}
          onClick={() => set('vibrate', !prefs.vibrate, { phone_drag_vibrate: !prefs.vibrate })}
        />
      </div>
    </section>
  )
}

/** Same switch as DayTasksSection's. */
function Switch({ on, label, hint, onClick, testId }: {
  on: boolean
  label: string
  hint: string
  onClick: () => void
  testId: string
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      data-testid={testId}
      onClick={onClick}
      className="w-full flex items-center justify-between gap-3 p-3 bg-zinc-800/50 rounded-lg hover:bg-zinc-800 transition-colors text-left"
    >
      <span>
        <span className="block text-sm text-zinc-300">{label}</span>
        <span className="block text-xs text-zinc-500 mt-0.5">{hint}</span>
      </span>
      <span className={`relative shrink-0 w-10 h-5 rounded-full transition-colors ${on ? 'bg-blue-600' : 'bg-zinc-600'}`}>
        <span
          className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
            on ? 'translate-x-5' : 'translate-x-0.5'
          }`}
        />
      </span>
    </button>
  )
}
