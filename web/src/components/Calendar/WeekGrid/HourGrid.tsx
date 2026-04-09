import { HOUR_PX } from './weekgrid.constants';

interface HourGridProps {
  isMobile: boolean;
}

export function HourGrid({ isMobile: _isMobile }: HourGridProps) {
  const hoursCount = 24;
  const labelInterval = 3;
  const lineInterval = 3;
  
  return (
    <>
      {Array.from({ length: hoursCount }, (_, h) => (
        <div
          key={h}
          className={`absolute left-0 right-0 border-t ${
            h % lineInterval === 0 ? 'border-zinc-700' : 'border-zinc-800/50'
          }`}
          style={{ top: h * HOUR_PX }}
        >
          {h % labelInterval === 0 && (
            <div className="absolute left-1 -top-2 px-1 text-[10px] text-zinc-500 bg-black font-mono">
              {String(h).padStart(2, '0')}:00
            </div>
          )}
        </div>
      ))}
    </>
  );
}
