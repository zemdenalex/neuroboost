import { minsToTop } from './weekgrid.utils';

interface TimeIndicatorProps {
  currentMinutes: number;
}

export function TimeIndicator({ currentMinutes }: TimeIndicatorProps) {
  if (currentMinutes < 0 || currentMinutes > 1440) {
    return null;
  }
  
  return (
    <div 
      className="absolute left-0 right-0 h-[2px] bg-red-500 z-30 pointer-events-none" 
      style={{ top: minsToTop(currentMinutes) }}
    >
      <div className="absolute -left-1 -top-[3px] w-2 h-2 rounded-full bg-red-500" />
      <div className="absolute left-2 -top-3 text-[10px] text-red-500 font-mono bg-black px-1">
        now
      </div>
    </div>
  );
}
