type RepeatType = 'none' | 'daily' | 'weekly' | 'monthly';
type RepeatEndType = 'never' | 'count' | 'date';

interface RepeatFieldsProps {
  repeatType: RepeatType;
  repeatEndType: RepeatEndType;
  repeatCount: number;
  repeatUntil: string;
  onRepeatTypeChange: (type: RepeatType) => void;
  onRepeatEndTypeChange: (type: RepeatEndType) => void;
  onRepeatCountChange: (count: number) => void;
  onRepeatUntilChange: (date: string) => void;
}

const REPEAT_OPTIONS: { value: RepeatType; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
];

const END_OPTIONS: { value: RepeatEndType; label: string }[] = [
  { value: 'never', label: 'Never' },
  { value: 'count', label: 'After N times' },
  { value: 'date', label: 'Until date' },
];

export function RepeatFields({
  repeatType,
  repeatEndType,
  repeatCount,
  repeatUntil,
  onRepeatTypeChange,
  onRepeatEndTypeChange,
  onRepeatCountChange,
  onRepeatUntilChange,
}: RepeatFieldsProps) {
  return (
    <div className="space-y-3">
      {/* Repeat frequency */}
      <div className="flex items-center gap-2 text-sm">
        <label className="text-zinc-400">Repeat:</label>
        <select
          value={repeatType}
          onChange={(e) => onRepeatTypeChange(e.target.value as RepeatType)}
          className="bg-zinc-800 border border-zinc-600 rounded px-2 py-1 text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
        >
          {REPEAT_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      {/* End condition (only when repeating) */}
      {repeatType !== 'none' && (
        <div className="ml-4 space-y-2 border-l-2 border-zinc-700 pl-3">
          <div className="flex items-center gap-2 text-sm">
            <label className="text-zinc-400">Ends:</label>
            <select
              value={repeatEndType}
              onChange={(e) => onRepeatEndTypeChange(e.target.value as RepeatEndType)}
              className="bg-zinc-800 border border-zinc-600 rounded px-2 py-1 text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
            >
              {END_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Count input */}
          {repeatEndType === 'count' && (
            <div className="flex items-center gap-2 text-sm">
              <label className="text-zinc-400">Occurrences:</label>
              <input
                type="number"
                min={2}
                max={365}
                value={repeatCount}
                onChange={(e) => {
                  const val = parseInt(e.target.value, 10);
                  if (!isNaN(val) && val >= 1) onRepeatCountChange(val);
                }}
                className="w-20 px-2 py-1 bg-zinc-800 border border-zinc-600 rounded text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
              />
            </div>
          )}

          {/* Until date input */}
          {repeatEndType === 'date' && (
            <div className="flex items-center gap-2 text-sm">
              <label className="text-zinc-400">Until:</label>
              <input
                type="date"
                value={repeatUntil}
                onChange={(e) => onRepeatUntilChange(e.target.value)}
                className="px-2 py-1 bg-zinc-800 border border-zinc-600 rounded text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
