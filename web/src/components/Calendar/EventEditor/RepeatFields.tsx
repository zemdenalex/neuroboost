import { useTranslation } from 'react-i18next';

type RepeatType = 'none' | 'daily' | 'weekly' | 'monthly' | 'yearly';
type RepeatEndType = 'never' | 'count' | 'date';

interface RepeatFieldsProps {
  repeatType: RepeatType;
  repeatEndType: RepeatEndType;
  repeatCount: number;
  repeatUntil: string;
  /** Every N days/weeks/months (row 11, 26.09); not shown for yearly. */
  repeatInterval: number;
  onRepeatIntervalChange: (n: number) => void;
  onRepeatTypeChange: (type: RepeatType) => void;
  onRepeatEndTypeChange: (type: RepeatEndType) => void;
  onRepeatCountChange: (count: number) => void;
  onRepeatUntilChange: (date: string) => void;
}

const REPEAT_TYPE_KEYS: Record<RepeatType, string> = {
  none: 'repeat.none',
  daily: 'repeat.daily',
  weekly: 'repeat.weekly',
  monthly: 'repeat.monthly',
  yearly: 'repeat.yearly',
};

const UNIT_KEYS: Partial<Record<RepeatType, string>> = {
  daily: 'repeat.unitDays',
  weekly: 'repeat.unitWeeks',
  monthly: 'repeat.unitMonths',
};

const END_TYPE_KEYS: Record<RepeatEndType, string> = {
  never: 'repeat.never',
  count: 'repeat.afterCount',
  date: 'repeat.untilDate',
};

export function RepeatFields({
  repeatType,
  repeatEndType,
  repeatCount,
  repeatUntil,
  repeatInterval,
  onRepeatIntervalChange,
  onRepeatTypeChange,
  onRepeatEndTypeChange,
  onRepeatCountChange,
  onRepeatUntilChange,
}: RepeatFieldsProps) {
  const { t } = useTranslation('calendar');

  return (
    <div className="space-y-3">
      {/* Repeat frequency */}
      <div className="flex items-center gap-2 text-sm">
        <label className="text-zinc-400">{t('repeat.repeat')}</label>
        <select
          value={repeatType}
          data-testid="event-repeat"
          onChange={(e) => {
            onRepeatTypeChange(e.target.value as RepeatType)
            // An interval means something else in another unit: start at 1.
            onRepeatIntervalChange(1)
          }}
          className="bg-zinc-800 border border-zinc-600 rounded px-2 py-1 text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
        >
          {(Object.keys(REPEAT_TYPE_KEYS) as RepeatType[]).map((key) => (
            <option key={key} value={key}>
              {t(REPEAT_TYPE_KEYS[key])}
            </option>
          ))}
        </select>
      </div>

      {/* End condition (only when repeating) */}
      {repeatType !== 'none' && (
        <div className="ml-4 space-y-2 border-l-2 border-zinc-700 pl-3">
          {UNIT_KEYS[repeatType] && (
            <div className="flex items-center gap-2 text-sm">
              <label htmlFor="event-repeat-interval" className="text-zinc-400">{t('repeat.every')}</label>
              <input
                id="event-repeat-interval"
                data-testid="event-repeat-interval"
                type="number"
                min={1}
                max={99}
                value={repeatInterval}
                onChange={(e) => {
                  const val = parseInt(e.target.value, 10);
                  if (!isNaN(val) && val >= 1 && val <= 99) onRepeatIntervalChange(val);
                }}
                className="w-16 px-2 py-1 bg-zinc-800 border border-zinc-600 rounded text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
              />
              <span className="text-zinc-400">{t(UNIT_KEYS[repeatType]!, { count: repeatInterval })}</span>
            </div>
          )}
          <div className="flex items-center gap-2 text-sm">
            <label className="text-zinc-400">{t('repeat.ends')}</label>
            <select
              value={repeatEndType}
              onChange={(e) => onRepeatEndTypeChange(e.target.value as RepeatEndType)}
              className="bg-zinc-800 border border-zinc-600 rounded px-2 py-1 text-white font-mono text-sm focus:outline-none focus:border-zinc-400"
            >
              {(Object.keys(END_TYPE_KEYS) as RepeatEndType[]).map((key) => (
                <option key={key} value={key}>
                  {t(END_TYPE_KEYS[key])}
                </option>
              ))}
            </select>
          </div>

          {/* Count input */}
          {repeatEndType === 'count' && (
            <div className="flex items-center gap-2 text-sm">
              <label className="text-zinc-400">{t('repeat.occurrences')}</label>
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
              <label className="text-zinc-400">{t('repeat.until')}</label>
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
