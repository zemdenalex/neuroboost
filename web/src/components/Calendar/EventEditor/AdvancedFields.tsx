import { useTranslation } from 'react-i18next'
import { PALETTE, PALETTE_NAMES, resolveColor } from '../../../lib/calendar/palette';
import { ReminderOffsets } from '../../ReminderOffsets/ReminderOffsets';
import type { ReminderPresets } from '../../../lib/reminders/offsets';

interface AdvancedFieldsProps {
  isAllDay: boolean;
  /**
   * Minutes before start, one entry per reminder. This replaced a single
   * `reminderMinutes` dropdown that sent a `reminders` array the Go API never
   * declared — the field was silently discarded, so that control had never
   * scheduled anything.
   */
  reminderOffsets: number[];
  presets: ReminderPresets;
  color: string;
  onAllDayChange: (value: boolean) => void;
  onReminderOffsetsChange: (value: number[]) => void;
  onColorChange: (value: string) => void;
}

export function AdvancedFields({
  isAllDay,
  reminderOffsets,
  presets,
  color,
  onAllDayChange,
  onReminderOffsetsChange,
  onColorChange,
}: AdvancedFieldsProps) {
  const { t } = useTranslation('calendar');

  return (
    <>
      <div className="flex items-center gap-4">
        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <input
            type="checkbox"
            checked={isAllDay}
            onChange={(e) => onAllDayChange(e.target.checked)}
            className="rounded bg-zinc-700 border-zinc-600 text-blue-500 focus:ring-blue-500 focus:ring-offset-zinc-900"
          />
          <span className="text-zinc-300">{t('advanced.allDay')}</span>
        </label>

      </div>

      <ReminderOffsets
        value={reminderOffsets}
        onChange={onReminderOffsetsChange}
        presets={presets}
      />

      <div>
        <label className="block text-xs text-zinc-400 mb-1">{t('advanced.color')}</label>
        {/* Swatches, not a text field.
            🔴 It was free text, and Denis reported that "blue" and "blue-400"
            did nothing — correctly, because a Tailwind class is not a CSS
            colour and the preview turned it into `var(--blue-400)`, which does
            not exist. A value the picker cannot show as selected is a value
            nobody can correct later, so the input now offers only colours this
            app can render. */}
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => onColorChange('')}
            aria-label={t('advanced.colorNone')}
            title={t('advanced.colorNone')}
            className={`w-6 h-6 rounded-full border grid place-items-center text-xs ${
              resolveColor(color) === undefined
                ? 'border-blue-400 ring-2 ring-blue-400/50 text-blue-300'
                : 'border-zinc-600 text-zinc-500 hover:border-zinc-400'
            }`}
          >
            ×
          </button>
          {PALETTE_NAMES.map((name) => {
            const hex = PALETTE[name]
            const selected = resolveColor(color) === hex
            return (
              <button
                key={name}
                type="button"
                onClick={() => onColorChange(name)}
                aria-label={name}
                title={name}
                className={`w-6 h-6 rounded-full border transition-transform ${
                  selected ? 'border-white ring-2 ring-white/60 scale-110' : 'border-zinc-600 hover:scale-105'
                }`}
                style={{ backgroundColor: hex }}
              />
            )
          })}
        </div>
        <div className="text-xs text-zinc-500 mt-1">
          {/* The event's own colour wins over its calendar's — stated here
              because the swatch row otherwise implies "no colour" means grey
              rather than "inherit from the calendar". */}
          {t('advanced.colorInherits')}
        </div>
      </div>
    </>
  );
}

/** Color presets for quick selection */
export const COLOR_PRESETS = [
  { name: 'Blue', value: '#3b82f6' },
  { name: 'Green', value: '#22c55e' },
  { name: 'Yellow', value: '#eab308' },
  { name: 'Red', value: '#ef4444' },
  { name: 'Purple', value: '#a855f7' },
  { name: 'Pink', value: '#ec4899' },
  { name: 'Orange', value: '#f97316' },
  { name: 'Teal', value: '#14b8a6' },
];
