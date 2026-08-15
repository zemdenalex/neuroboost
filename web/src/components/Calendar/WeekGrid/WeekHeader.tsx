import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { getTimezoneOffsetMs } from './weekgrid.utils';
import { DAY_MS } from './weekgrid.constants';
import { dateLocale } from '../../../utils/date';

interface WeekHeaderProps {
  mondayUtc0: number;
  visibleDays: number;
  currentWeekOffset: number;
  timezone: string;
  isMobile: boolean;
  onWeekChange?: (offset: number) => void;
  onMobileNav?: (direction: 'prev' | 'next') => void;
  onToday?: () => void;
  onQuickCreate: () => void;
  /** Extra controls for the right-hand button row (the calendar filter). */
  headerExtra?: ReactNode;
}

export function WeekHeader({
  mondayUtc0,
  visibleDays,
  currentWeekOffset,
  timezone,
  isMobile,
  onWeekChange,
  onMobileNav,
  onToday,
  onQuickCreate,
  headerExtra,
}: WeekHeaderProps) {
  const { t, i18n } = useTranslation('calendar');
  const locale = dateLocale(i18n.language);

  const offset = getTimezoneOffsetMs(timezone);
  const startDate = new Date(mondayUtc0 + offset);
  const endDate = new Date(mondayUtc0 + 6 * DAY_MS + offset);

  const weekLabel = (() => {
    if (visibleDays === 1) {
      return startDate.toLocaleDateString(locale, {
        weekday: 'long',
        month: 'long',
        day: 'numeric'
      });
    } else if (visibleDays === 3) {
      const rangeEnd = new Date(mondayUtc0 + (visibleDays - 1) * DAY_MS + offset);
      return `${startDate.toLocaleDateString(locale, { month: 'short', day: 'numeric' })} – ${rangeEnd.toLocaleDateString(locale, { month: 'short', day: 'numeric' })}`;
    }
    return `${startDate.toLocaleDateString(locale, { month: 'short', day: 'numeric' })} – ${endDate.toLocaleDateString(locale, { month: 'short', day: 'numeric', year: 'numeric' })}`;
  })();

  return (
    <div className="px-2 py-2 border-b border-zinc-700 bg-zinc-900">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          {(onWeekChange || onMobileNav) && (
            <>
              <button
                onClick={() => {
                  if (onMobileNav) {
                    onMobileNav('prev');
                  } else if (onWeekChange) {
                    onWeekChange(currentWeekOffset - 1);
                  }
                }}
                className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 transition-colors"
                title="Previous period"
              >
                ←
              </button>
              <button
                onClick={() => {
                  if (onMobileNav) {
                    onMobileNav('next');
                  } else if (onWeekChange) {
                    onWeekChange(currentWeekOffset + 1);
                  }
                }}
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
          {(currentWeekOffset !== 0 || onMobileNav) && (onWeekChange || onToday) && (
            <button
              onClick={() => {
                if (onToday) {
                  onToday();
                } else if (onWeekChange) {
                  onWeekChange(0);
                }
              }}
              className="px-2 py-1 text-xs rounded bg-zinc-700 hover:bg-zinc-600 text-white transition-colors"
            >
              {t('today')}
            </button>
          )}
          {!isMobile && (
            <button
              data-hint="calendar.newEvent"
              onClick={onQuickCreate}
              className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 transition-colors"
            >
              + {t('quickCreate')}
            </button>
          )}
          {/* Last on purpose. This slot holds the calendar filter, whose panel
              is `absolute right-0` — anchored to its own button, not to the
              screen. Sitting before the Today button put that anchor 62px in
              from the right edge, so a 320px panel started at x = -13 and ran
              off a 375px screen. Anything added after this will push it back
              off; add it before, not after. Measured by
              web/e2e/mobile-overflow.spec.ts. */}
          {headerExtra}
        </div>
      </div>
      
      <div className="text-xs text-zinc-400">
        {isMobile ? t('helpMobile') : t('helpDesktop')}
      </div>
    </div>
  );
}
