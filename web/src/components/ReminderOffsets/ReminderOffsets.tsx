import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { presetLabel } from '../../lib/reminders/presetLabel'
import { Bell, BellOff, Plus, X } from 'lucide-react'
import {
  addOffset,
  formatOffset,
  matchPreset,
  parseOffsetInput,
  removeOffset,
  type ReminderPresets,
} from '../../lib/reminders/offsets'

interface ReminderOffsetsProps {
  value: number[]
  onChange: (offsets: number[]) => void
  presets: ReminderPresets
  /** Disabled when a task has no due date — there is nothing to count back from. */
  disabled?: boolean
  disabledHint?: string
  /**
   * Show the "apply a preset" dropdown. Default true.
   *
   * 🔴 Must be false when this component is editing a PRESET ITSELF, which is
   * what the Settings presets list does. Otherwise each preset row carries a
   * control that overwrites that preset with a different one's offsets — and
   * that is not hypothetical: on 2026-08-14 the staging account held
   * {"без": [1440,60], "важное": [1440,60], "обычное": [1440,60]}, all three
   * flattened to the same values by exactly this control.
   *
   * The damage compounds, because matchPreset returns the FIRST preset whose
   * offsets match. Once two coincide, the dropdown reports the wrong name, and
   * choosing another preset writes identical offsets and snaps straight back —
   * which reads as "the selector is broken".
   */
  showPresetPicker?: boolean
}

/**
 * The one editor for reminder offsets, used in the event editor, the task
 * editor and Settings. Offsets are a list rather than a single dropdown
 * because one event legitimately wants five reminders and another wants one.
 */
export function ReminderOffsets({
  value,
  onChange,
  presets,
  disabled = false,
  disabledHint,
  showPresetPicker = true,
}: ReminderOffsetsProps) {
  const { t } = useTranslation('reminders')
  const [draft, setDraft] = useState('')
  const [error, setError] = useState<string | null>(null)

  const activePreset = matchPreset(value, presets)

  const label = (minutes: number) => {
    const { value: amount, unit } = formatOffset(minutes)
    if (unit === 'atStart') return t('offset.atStart')
    return t(`offset.${unit}`, { count: amount })
  }

  const commitDraft = () => {
    const parsed = parseOffsetInput(draft)
    if (parsed === null) {
      setError(t('add.invalid'))
      return
    }
    onChange(addOffset(value, parsed))
    setDraft('')
    setError(null)
  }

  if (disabled) {
    return (
      <div className="flex items-center gap-2 text-sm text-zinc-500">
        <BellOff size={16} />
        <span>{disabledHint ?? t('disabled')}</span>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <span className="flex items-center gap-2 text-sm text-zinc-400">
          <Bell size={16} />
          {t('title')}
        </span>
        {showPresetPicker && (
          <select
            aria-label={t('preset.label')}
            className="bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-white"
            value={activePreset ?? ''}
            onChange={e => {
              const preset = presets[e.target.value]
              if (preset) onChange([...preset])
            }}
          >
            {/* An empty option only exists while the list matches no preset, so
                the select never silently reports a preset the user is not on. */}
            {activePreset === null && <option value="">{t('preset.custom')}</option>}
            {Object.keys(presets).map(name => (
              <option key={name} value={name}>{presetLabel(name, t)}</option>
            ))}
          </select>
        )}
      </div>

      {value.length === 0 ? (
        <p className="text-sm text-zinc-500">{t('empty')}</p>
      ) : (
        <ul className="flex flex-wrap gap-2">
          {value.map(minutes => (
            <li key={minutes}>
              <span className="inline-flex items-center gap-1 bg-zinc-800 border border-zinc-700 rounded-full pl-3 pr-1 py-1 text-sm text-zinc-200">
                {label(minutes)}
                <button
                  type="button"
                  aria-label={t('remove', { offset: label(minutes) })}
                  className="p-1 rounded-full text-zinc-400 hover:text-white hover:bg-zinc-700"
                  onClick={() => onChange(removeOffset(value, minutes))}
                >
                  <X size={14} />
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      <div className="flex items-center gap-2">
        <input
          type="text"
          inputMode="text"
          value={draft}
          placeholder={t('add.placeholder')}
          aria-label={t('add.label')}
          aria-invalid={error !== null}
          className="flex-1 bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-white placeholder-zinc-600"
          onChange={e => { setDraft(e.target.value); setError(null) }}
          onKeyDown={e => {
            // Enter adds an offset and stays put. It must not reach the
            // surrounding form, or typing "30" into a reminder would submit
            // the whole event.
            if (e.key === 'Enter') {
              e.preventDefault()
              e.stopPropagation()
              commitDraft()
            }
          }}
        />
        <button
          type="button"
          className="inline-flex items-center gap-1 bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-zinc-200 hover:bg-zinc-700 disabled:opacity-40"
          disabled={draft.trim() === ''}
          onClick={commitDraft}
        >
          <Plus size={14} />
          {t('add.button')}
        </button>
      </div>
      {error && <p role="alert" className="text-sm text-red-400">{error}</p>}
    </div>
  )
}
