import { useTranslation } from 'react-i18next';
import { Users } from 'lucide-react';
import { ALL_DAY_HEIGHT, DAY_MS } from './weekgrid.constants';
import type { ProcessedEvent, DragState, NbEvent, DayInfo } from './weekgrid.types';
import { getTimezoneOffsetMs, formatAllDayGhostLabel } from './weekgrid.utils';
import { resolveColor } from '../../../lib/calendar/palette';

interface AllDaySectionProps {
  days: DayInfo[];
  allDayEvents: ProcessedEvent[];
  selectedId: string | null;
  drag: DragState;
  mondayUtc0: number;
  visibleDays: number;
  timezone: string;
  onSelect: (event: NbEvent) => void;
  onSelectId: (id: string) => void;
  onDragStart: (dayUtc0: number, eventId?: string, span?: number) => void;
}

export function AllDaySection({
  days,
  allDayEvents,
  selectedId,
  drag,
  mondayUtc0,
  visibleDays,
  timezone,
  onSelect,
  onSelectId,
  onDragStart,
}: AllDaySectionProps) {
  const { t, i18n } = useTranslation('calendar');
  const offset = getTimezoneOffsetMs(timezone);

  return (
    <div 
      className="col-span-full sticky top-0 z-30 bg-zinc-900 border-b border-zinc-700"
      style={{ height: ALL_DAY_HEIGHT }}
    >
      <div className={`grid gap-px h-full ${
        visibleDays === 1 ? 'grid-cols-1' : 
        visibleDays === 3 ? 'grid-cols-3' : 
        'grid-cols-7'
      }`}>
        {days.map(({ i, dayUtc0 }) => {
          const dayEvents = allDayEvents.filter(e => 
            e.span && e.span.startDay <= i && i <= e.span.endDay
          );
          
          return (
            <div 
              key={`allday-${i}`}
              className="relative border-r border-zinc-700 last:border-r-0"
              onMouseDown={() => onDragStart(dayUtc0)}
            >
              <div className="absolute inset-0 flex items-start p-1">
                <span className="text-xs text-zinc-500 font-mono">{t('allDay')}</span>
              </div>
              
              {dayEvents.map(e => {
                const selected = selectedId === e.id;
                const isStartDay = e.span!.startDay === i;
                const isEndDay = e.span!.endDay === i;
                
                return (
                  <div
                    key={`allday-event-${e.id}-${i}`}
                    className={`absolute top-5 bottom-2 font-mono text-xs px-2 flex items-center cursor-pointer
                      ${selected 
                        ? 'ring-1 ring-blue-400 bg-blue-600/90 z-40' 
                        : 'bg-zinc-600/90 hover:bg-zinc-500/90 z-25'}
                      border border-zinc-500`}
                    style={{
                      left: isStartDay ? 4 : 0,
                      right: isEndDay ? 4 : 0,
                      borderRadius: isStartDay && isEndDay ? '6px' :
                                  isStartDay ? '6px 0 0 6px' :
                                  isEndDay ? '0 6px 6px 0' : '0',
                      // displayColor, not e.color: all-day rows get one from
                      // processEventsForWeek like every other event, and it is
                      // the only value that carries the calendar's colour AND
                      // is resolved to paintable CSS. Reading e.color here meant
                      // an all-day event ignored its calendar entirely and
                      // painted nothing for `blue-400`.
                      backgroundColor: e.displayColor ?? resolveColor(e.color),
                    }}
                    onClick={(ev) => {
                      ev.stopPropagation();
                      if (e.id) onSelectId(e.id);
                    }}
                    onDoubleClick={() => onSelect(e)}
                    onMouseDown={(ev) => {
                      if (!e.id) return;
                      ev.stopPropagation();
                      onDragStart(dayUtc0, e.id, e.span?.spanDays);
                    }}
                    title={`${e.title} • All-day event`}
                  >
                    {isStartDay && (
                      // 🔴 The second place an event is drawn. The calendar
                      // colour bug lived here for exactly this reason: work
                      // done in EventBlock was assumed to cover the grid, and
                      // this strip was never opened. Whatever an event block
                      // shows about sharing, this must show too.
                      <span className="break-words text-white font-medium inline-flex items-center gap-1 min-w-0">
                        <span className="min-w-0 break-words">{e.title || '(untitled)'}</span>
                        {e.isShared && (
                          <Users
                            size={11}
                            aria-label="Общий календарь"
                            data-testid="shared-badge"
                            className="text-zinc-200 flex-shrink-0"
                          />
                        )}
                        {e.authorName && (
                          <span className="min-w-0 truncate text-zinc-300 font-normal" data-testid="event-author">
                            · {e.authorName}
                          </span>
                        )}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          );
        })}
      </div>
      
      {/* Multi-day create ghost in all-day section */}
      {drag && drag.kind === 'create' && drag.allDay && drag.crossDay && (
        <AllDayCreateGhost
          drag={drag}
          mondayUtc0={mondayUtc0}
          visibleDays={visibleDays}
          offset={offset}
          language={i18n.language}
        />
      )}

      {/* Single-day all-day create ghost */}
      {drag && drag.kind === 'create' && drag.allDay && !drag.crossDay && (
        <SingleAllDayGhost
          drag={drag}
          mondayUtc0={mondayUtc0}
          visibleDays={visibleDays}
          offset={offset}
          language={i18n.language}
        />
      )}
    </div>
  );
}

function AllDayCreateGhost({
  drag,
  mondayUtc0,
  visibleDays,
  offset,
  language,
}: {
  drag: NonNullable<DragState> & { kind: 'create' };
  mondayUtc0: number;
  visibleDays: number;
  offset: number;
  language: string;
}) {
  const startDayIndex = Math.max(0, Math.min(visibleDays - 1, 
    Math.floor((Math.min(drag.startDayUtc0, drag.endDayUtc0) - mondayUtc0) / DAY_MS)));
  const endDayIndex = Math.max(0, Math.min(visibleDays - 1, 
    Math.floor((Math.max(drag.startDayUtc0, drag.endDayUtc0) - mondayUtc0) / DAY_MS)));
  
  const leftPercent = (startDayIndex / visibleDays) * 100;
  const widthPercent = ((endDayIndex - startDayIndex + 1) / visibleDays) * 100;
  
  const label = formatAllDayGhostLabel(drag.startDayUtc0, drag.endDayUtc0, offset, language);
  
  return (
    <div 
      className="absolute bg-emerald-400/40 border border-emerald-400/60 pointer-events-none transition-all duration-150 flex items-center justify-center"
      style={{
        top: 25,
        height: ALL_DAY_HEIGHT - 30,
        left: `calc(${leftPercent}% + 4px)`,
        width: `calc(${widthPercent}% - 8px)`,
        borderRadius: '6px',
        zIndex: 50
      }}
    >
      <span className="text-xs font-mono text-emerald-100 bg-emerald-800/90 px-2 py-1 rounded">
        {label}
      </span>
    </div>
  );
}

function SingleAllDayGhost({ drag, mondayUtc0, visibleDays, offset, language }: {
  drag: NonNullable<DragState> & { kind: 'create' }; mondayUtc0: number; visibleDays: number; offset: number; language: string;
}) {
  const dayIndex = Math.max(0, Math.min(visibleDays - 1, Math.floor((drag.startDayUtc0 - mondayUtc0) / DAY_MS)));
  const label = formatAllDayGhostLabel(drag.startDayUtc0, drag.startDayUtc0, offset, language);
  
  return (
    <div className="absolute bg-emerald-400/40 border border-emerald-400/60 pointer-events-none flex items-center justify-center"
      style={{ top: 25, height: ALL_DAY_HEIGHT - 30, left: `calc(${(dayIndex / visibleDays) * 100}% + 4px)`,
        width: `calc(${(1 / visibleDays) * 100}% - 8px)`, borderRadius: '6px', zIndex: 50 }}>
      <span className="text-xs font-mono text-emerald-100 bg-emerald-800/90 px-1 rounded">{label}</span>
    </div>
  );
}
