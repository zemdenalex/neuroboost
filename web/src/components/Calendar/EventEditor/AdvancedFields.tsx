import { useTranslation } from 'react-i18next';
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
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder={t('advanced.colorPlaceholder')}
            value={color}
            onChange={(e) => onColorChange(e.target.value)}
            className="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-600 rounded text-white placeholder-zinc-400 focus:outline-none focus:border-zinc-400 font-mono"
          />
          {color && (
            <div
              className="w-8 h-8 rounded border border-zinc-600"
              style={{ backgroundColor: color.startsWith('#') ? color : `var(--${color})` }}
            />
          )}
        </div>
        <div className="text-xs text-zinc-500 mt-1">
          {t('colorHint')}
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
