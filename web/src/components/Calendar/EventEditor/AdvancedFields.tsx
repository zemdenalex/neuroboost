import { REMINDER_OPTIONS } from './editor.types';

interface AdvancedFieldsProps {
  isAllDay: boolean;
  reminderMinutes: number;
  color: string;
  onAllDayChange: (value: boolean) => void;
  onReminderChange: (value: number) => void;
  onColorChange: (value: string) => void;
}

export function AdvancedFields({
  isAllDay,
  reminderMinutes,
  color,
  onAllDayChange,
  onReminderChange,
  onColorChange,
}: AdvancedFieldsProps) {
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
          <span className="text-zinc-300">All Day</span>
        </label>

        <div className="flex items-center gap-2 text-sm">
          <label className="text-zinc-400">Reminder:</label>
          <select
            value={reminderMinutes}
            onChange={(e) => onReminderChange(Number(e.target.value))}
            className="bg-zinc-800 border border-zinc-600 rounded px-2 py-1 text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
          >
            {REMINDER_OPTIONS.map(opt => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div>
        <label className="block text-xs text-zinc-400 mb-1">Color (optional)</label>
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder="#3b82f6 or blue-500"
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
          Hex color (#ff0000) or Tailwind class (blue-500)
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
