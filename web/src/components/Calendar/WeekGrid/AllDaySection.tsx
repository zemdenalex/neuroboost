import { ALL_DAY_HEIGHT, DAY_MS } from './weekgrid.constants';
import type { ProcessedEvent, DragState, NbEvent, DayInfo } from './weekgrid.types';
import { getTimezoneOffsetMs, formatDayLabel } from './weekgrid.utils';

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
                <span className="text-xs text-zinc-500 font-mono">all-day</span>
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
                      backgroundColor: e.color || undefined,
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
                      <span className="break-words text-white font-medium">
                        {e.title || '(untitled)'}
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
        />
      )}
      
      {/* Single-day all-day create ghost */}
      {drag && drag.kind === 'create' && drag.allDay && !drag.crossDay && (
        <SingleAllDayGhost 
          drag={drag} 
          mondayUtc0={mondayUtc0} 
          visibleDays={visibleDays}
          offset={offset}
        />
      )}
    </div>
  );
}

function AllDayCreateGhost({ 
  drag, 
  mondayUtc0, 
  visibleDays,
  offset 
}: { 
  drag: NonNullable<DragState> & { kind: 'create' }; 
  mondayUtc0: number; 
  visibleDays: number;
  offset: number;
}) {
  const startDayIndex = Math.max(0, Math.min(visibleDays - 1, 
    Math.floor((Math.min(drag.startDayUtc0, drag.endDayUtc0) - mondayUtc0) / DAY_MS)));
  const endDayIndex = Math.max(0, Math.min(visibleDays - 1, 
    Math.floor((Math.max(drag.startDayUtc0, drag.endDayUtc0) - mondayUtc0) / DAY_MS)));
  
  const leftPercent = (startDayIndex / visibleDays) * 100;
  const widthPercent = ((endDayIndex - startDayIndex + 1) / visibleDays) * 100;
  
  const startDate = new Date(Math.min(drag.startDayUtc0, drag.endDayUtc0) + offset);
  const endDate = new Date(Math.max(drag.startDayUtc0, drag.endDayUtc0) + offset);
  const label = `${startDate.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' })} — ${endDate.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' })}`;
  
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

function SingleAllDayGhost({ drag, mondayUtc0, visibleDays, offset }: { 
  drag: NonNullable<DragState> & { kind: 'create' }; mondayUtc0: number; visibleDays: number; offset: number;
}) {
  const dayIndex = Math.max(0, Math.min(visibleDays - 1, Math.floor((drag.startDayUtc0 - mondayUtc0) / DAY_MS)));
  const label = new Date(drag.startDayUtc0 + offset).toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' });
  
  return (
    <div className="absolute bg-emerald-400/40 border border-emerald-400/60 pointer-events-none flex items-center justify-center"
      style={{ top: 25, height: ALL_DAY_HEIGHT - 30, left: `calc(${(dayIndex / visibleDays) * 100}% + 4px)`,
        width: `calc(${(1 / visibleDays) * 100}% - 8px)`, borderRadius: '6px', zIndex: 50 }}>
      <span className="text-xs font-mono text-emerald-100 bg-emerald-800/90 px-1 rounded">{label}</span>
    </div>
  );
}
