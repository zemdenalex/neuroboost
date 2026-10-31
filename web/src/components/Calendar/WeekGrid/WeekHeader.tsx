import { useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { getTimezoneOffsetMs } from './weekgrid.utils';
import { DAY_MS } from './weekgrid.constants';
import { dateLocale } from '../../../utils/date';
import { takeCalendarHint } from '../../../lib/calendar/calendarChrome';

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
  /**
   * Header C on a phone (Denis 31.10): the title is the month's name and opens
   * the month; the week strip below is the navigation, so no arrows here.
   */
  onOpenMonth?: () => void;
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
  onOpenMonth,
  headerExtra,
}: WeekHeaderProps) {
  const { t, i18n } = useTranslation('calendar');
  // On a phone the tip shows on the first three opens, then lives under «?»
  // (Denis 25.09, variant A). Counted once per mount, not per render.
  const [mobileHint] = useState(() =>
    // Accessors, not window.localStorage itself: touching it can throw when
    // storage is blocked, and the throw must land inside takeCalendarHint's try.
    takeCalendarHint({
      getItem: (k) => window.localStorage.getItem(k),
      setItem: (k, v) => window.localStorage.setItem(k, v),
    })
  );
  const locale = dateLocale(i18n.language);

  const offset = getTimezoneOffsetMs(timezone);
  const startDate = new Date(mondayUtc0 + offset);
  const endDate = new Date(mondayUtc0 + 6 * DAY_MS + offset);

  const weekLabel = (() => {
    if (visibleDays === 1) {
      // Short on a phone: beside the arrows and buttons the long form
      // ("Friday, September 25") broke onto two lines (mobile tour 25.09, MW4).
      return startDate.toLocaleDateString(locale, {
        weekday: isMobile ? 'short' : 'long',
        month: isMobile ? 'short' : 'long',
        day: 'numeric'
      });
    } else if (visibleDays === 3) {
      const rangeEnd = new Date(mondayUtc0 + (visibleDays - 1) * DAY_MS + offset);
      return `${startDate.toLocaleDateString(locale, { month: 'short', day: 'numeric' })} – ${rangeEnd.toLocaleDateString(locale, { month: 'short', day: 'numeric' })}`;
    }
    return `${startDate.toLocaleDateString(locale, { month: 'short', day: 'numeric' })} – ${endDate.toLocaleDateString(locale, { month: 'short', day: 'numeric', year: 'numeric' })}`;
  })();

  // Header C: «Октябрь ▾». Month name of the day on screen, standalone form
  // (ru «октябрь», not the genitive «октября» a date uses).
  const monthTitle = startDate.toLocaleDateString(locale, { month: 'long' });

  const arrow = (dir: 'prev' | 'next') => (
    <button
      onClick={() => {
        if (onMobileNav) {
          onMobileNav(dir);
        } else if (onWeekChange) {
          onWeekChange(currentWeekOffset + (dir === 'next' ? 1 : -1));
        }
      }}
      className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 transition-colors"
      title={dir === 'next' ? 'Next period' : 'Previous period'}
    >
      {isMobile ? (dir === 'next' ? '›' : '‹') : (dir === 'next' ? '→' : '←')}
    </button>
  );

  const showHint = !isMobile || mobileHint;

  return (
    <div data-testid="calendar-header" className="px-2 py-2 border-b border-zinc-700 bg-zinc-900">
      {/* Wraps rather than overflows: on a 320px phone arrows, title, Today,
          + Task and the calendar filter do not fit on one line and the last
          buttons ran off the screen (26.09). The right group keeps ml-auto so
          a wrapped row still ends at the right edge, where the filter panel
          (absolute right-0) expects its anchor. */}
      <div className={`flex flex-wrap items-center justify-between gap-y-2 ${showHint ? 'mb-2' : ''}`}>
        {/* Phone (header A, Denis 31.10): ‹ date › on one line with the
            buttons; the desktop keeps ← → before the title. */}
        <div className="flex items-center gap-2 min-w-0">
          {onOpenMonth ? (
            <button
              type="button"
              data-testid="calendar-month-title"
              onClick={onOpenMonth}
              className="font-semibold text-sm capitalize whitespace-nowrap"
            >
              {monthTitle} ▾
            </button>
          ) : (
            <>
              {(onWeekChange || onMobileNav) && arrow('prev')}
              {!isMobile && (onWeekChange || onMobileNav) && arrow('next')}
              <h2 data-testid="calendar-period-title" className="whitespace-nowrap font-semibold text-sm md:text-lg">{weekLabel}</h2>
              {isMobile && (onWeekChange || onMobileNav) && arrow('next')}
            </>
          )}
        </div>

        <div className="ml-auto flex items-center gap-1.5 md:gap-2">
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
              {isMobile ? t('todayShort') : t('today')}
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
      
      {showHint && (
        <div data-testid="calendar-hint" className="text-xs text-zinc-400 truncate">
          {isMobile ? t('helpMobile') : t('helpDesktop')}
        </div>
      )}
    </div>
  );
}
