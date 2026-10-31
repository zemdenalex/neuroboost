import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Smartphone } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import type { UserSettings } from '../../../api/auth'
import {
  ADD_MODES,
  PHONE_HEADERS,
  readAddMode,
  readPhoneHeader,
  type AddMode,
  type PhoneHeader,
} from '../../../lib/calendar/phoneCalendar'

interface Props {
  /** The page's debounced saver (see UIScaleSection for why it is passed in). */
  autoSave: (patch: Partial<UserSettings>) => void
}

/** A static sketch of each header: enough to tell them apart. */
function HeaderSketch({ v }: { v: PhoneHeader }) {
  if (v === 'row') {
    return (
      <div className="flex flex-col gap-0.5">
        <div className="flex gap-0.5 items-center">
          <div className="h-3 w-3 rounded-sm bg-zinc-700" />
          <div className="h-2 w-10 rounded-sm bg-zinc-500" />
          <div className="h-3 w-3 rounded-sm bg-zinc-700" />
          <div className="ml-auto h-3 w-6 rounded-sm bg-zinc-700" />
          <div className="h-3 w-5 rounded-sm bg-zinc-600" />
        </div>
        <div className="h-3 w-full border-l-2 border-purple-500 bg-zinc-800" />
        <div className="h-3 w-2/3 border-l-2 border-green-500 bg-zinc-800" />
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-0.5">
      <div className="h-2 w-12 rounded-sm bg-zinc-500" />
      <div className="grid grid-cols-7 gap-px">
        {[40, 80, 0, 100, 30, 60, 10].map((w, i) => (
          <div key={i} className={`h-4 rounded-sm flex items-end px-px pb-0.5 ${i === 3 ? 'bg-zinc-700' : 'bg-black border border-zinc-800'}`}>
            <div className="h-0.5 bg-blue-500" style={{ width: `${w}%` }} />
          </div>
        ))}
      </div>
      <div className="h-3 w-full border-l-2 border-purple-500 bg-zinc-800" />
    </div>
  )
}

/**
 * The phone calendar's header and its «+» (Denis 31.10, spec
 * V003-20261031-arc-shapka-dobavit). Shown on the computer too: the choice is
 * for the phone, and it is made wherever the person happens to be.
 */
export function PhoneCalendarSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [header, setHeader] = useState<PhoneHeader>(() => readPhoneHeader(user?.settings))
  const [mode, setMode] = useState<AddMode>(() => readAddMode(user?.settings))

  // Keyed on the account, not the user object: see UIScaleSection.
  useEffect(() => {
    setHeader(readPhoneHeader(user?.settings))
    setMode(readAddMode(user?.settings))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const card = (on: boolean) =>
    `text-left p-2 rounded-lg border transition-colors flex flex-col gap-2 min-w-0 ${
      on ? 'bg-blue-600/20 border-blue-500' : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
    }`

  return (
    <section data-testid="phone-calendar-section" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-1">
        <Smartphone className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('phoneCalendar.title')}</h2>
      </div>
      <p className="text-xs text-zinc-500 mb-3">{t('phoneCalendar.note')}</p>

      <h3 className="text-sm font-mono font-semibold text-zinc-200 mb-2">{t('phoneCalendar.header')}</h3>
      <div role="radiogroup" aria-label={t('phoneCalendar.header')} className="grid grid-cols-2 gap-2 md:w-[24rem]">
        {PHONE_HEADERS.map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={header === v}
            data-testid={`phone-header-${v}`}
            onClick={() => {
              setHeader(v)
              autoSave({ phone_header_variant: v })
            }}
            className={card(header === v)}
          >
            <HeaderSketch v={v} />
            <span className={`text-sm font-mono ${header === v ? 'text-blue-300' : 'text-zinc-200'}`}>
              {t(`phoneCalendar.${v}.name`)}
            </span>
            <span className="text-xs text-zinc-500">{t(`phoneCalendar.${v}.desc`)}</span>
          </button>
        ))}
      </div>

      <h3 className="mt-5 text-sm font-mono font-semibold text-zinc-200 mb-2">{t('phoneCalendar.add')}</h3>
      <div role="radiogroup" aria-label={t('phoneCalendar.add')} className="grid grid-cols-2 gap-2 md:w-[24rem]">
        {ADD_MODES.map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={mode === v}
            data-testid={`add-mode-${v}`}
            onClick={() => {
              setMode(v)
              autoSave({ add_mode: v })
            }}
            className={card(mode === v)}
          >
            <span className={`text-sm font-mono ${mode === v ? 'text-blue-300' : 'text-zinc-200'}`}>
              {t(`phoneCalendar.${v}.name`)}
            </span>
            <span className="text-xs text-zinc-500">{t(`phoneCalendar.${v}.desc`)}</span>
          </button>
        ))}
      </div>
    </section>
  )
}
