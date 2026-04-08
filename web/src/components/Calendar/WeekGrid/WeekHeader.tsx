import { useTranslation } from 'react-i18next';
import { getTimezoneOffsetMs } from './weekgrid.utils';
import { DAY_MS } from './weekgrid.constants';

interface WeekHeaderProps {
  mondayUtc0: number;
  visibleDays: number;
  currentWeekOffset: number;
  timezone: string;
  isMobile: boolean;
  onWeekChange?: (offset: number) => void;
  onQuickCreate: () => void;
}

export function WeekHeader({
  mondayUtc0,
  visibleDays,
  currentWeekOffset,
  timezone,
  isMobile,
  onWeekChange,
  onQuickCreate,
}: WeekHeaderProps) {
  const { t, i18n } = useTranslation('calendar');
  const dateLocale = i18n.language === 'ru' ? 'ru-RU' : 'en-US';

  const offset = getTimezoneOffsetMs(timezone);
  const startDate = new Date(mondayUtc0 + offset);
  const endDate = new Date(mondayUtc0 + 6 * DAY_MS + offset);

  const weekLabel = (() => {
    if (visibleDays === 1) {
      return startDate.toLocaleDateString(dateLocale, {
        weekday: 'long',
        month: 'long',
        day: 'numeric'
      });
    } else if (visibleDays === 3) {
      return `${startDate.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric' })} – ${endDate.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric' })}`;
    }
    return `${startDate.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric' })} – ${endDate.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric', year: 'numeric' })}`;
  })();

  return (
    <div className="px-2 py-2 border-b border-zinc-700 bg-zinc-900">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          {onWeekChange && (
            <>
              <button
                onClick={() => onWeekChange(currentWeekOffset - 1)}
                className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 transition-colors"
                title="Previous period"
              >
                ←
              </button>
              <button
                onClick={() => onWeekChange(currentWeekOffset + 1)}
                className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 transition-colors"
                title="Next period"
              >
                →
              </button>
            </>
          )}
          <h2 className="font-semibold text-sm md:text-lg">{weekLabel}</h2>
        </div>
        
        <div className="flex items-center gap-2">
          {currentWeekOffset !== 0 && onWeekChange && (
            <button
              onClick={() => onWeekChange(0)}
              className="px-2 py-1 text-xs rounded bg-zinc-700 hover:bg-zinc-600 text-white transition-colors"
            >
              {t('today')}
            </button>
          )}
          {!isMobile && (
            <button
              onClick={onQuickCreate}
              className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 transition-colors"
            >
              + {t('quickCreate')}
            </button>
          )}
        </div>
      </div>
      
      <div className="text-xs text-zinc-400">
        {isMobile ? t('helpMobile') : t('helpDesktop')}
      </div>
    </div>
  );
}
