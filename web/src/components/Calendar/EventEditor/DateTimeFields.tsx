import { TimeInput } from './TimeInput';
import type { TimeValidation } from './editor.types';

interface DateTimeFieldsProps {
  startDate: string;
  endDate: string;
  startTime: string;
  endTime: string;
  validation: TimeValidation;
  timezone: string;
  onStartDateChange: (value: string) => void;
  onEndDateChange: (value: string) => void;
  onStartTimeChange: (value: string, parsed: string | null) => void;
  onEndTimeChange: (value: string, parsed: string | null) => void;
  onStartTimeEnter?: () => void;
  onEndTimeEnter?: () => void;
}

export function DateTimeFields({
  startDate,
  endDate,
  startTime,
  endTime,
  validation,
  timezone,
  onStartDateChange,
  onEndDateChange,
  onStartTimeChange,
  onEndTimeChange,
  onStartTimeEnter,
  onEndTimeEnter,
}: DateTimeFieldsProps) {
  const isCrossMidnight = 
    validation.start && 
    validation.end && 
    validation.startParsed && 
    validation.endParsed &&
    startDate === endDate && 
    validation.endParsed < validation.startParsed &&
    validation.dateRangeValid;

  return (
    <div className="space-y-3">
      {/* Date inputs */}
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="block text-xs text-zinc-400 mb-1">Start Date</label>
          <input
            type="date"
            value={startDate}
            onChange={(e) => onStartDateChange(e.target.value)}
            className="w-full px-2 py-1 bg-zinc-800 border border-zinc-600 rounded text-white text-sm focus:outline-none focus:border-zinc-400"
          />
        </div>
        <div>
          <label className="block text-xs text-zinc-400 mb-1">End Date</label>
          <input
            type="date"
            value={endDate}
            onChange={(e) => onEndDateChange(e.target.value)}
            className="w-full px-2 py-1 bg-zinc-800 border border-zinc-600 rounded text-white text-sm focus:outline-none focus:border-zinc-400"
          />
        </div>
      </div>
      
      {/* Time inputs */}
      <div className="grid grid-cols-2 gap-2">
        <TimeInput
          value={startTime}
          onChange={onStartTimeChange}
          onEnter={onStartTimeEnter}
          label="Start Time"
          timezone={timezone}
        />
        <TimeInput
          value={endTime}
          onChange={onEndTimeChange}
          onEnter={onEndTimeEnter}
          label="End Time"
          timezone={timezone}
        />
      </div>
      
      {/* Validation error */}
      {!validation.dateRangeValid && validation.dateRangeError && (
        <div className="text-xs text-red-400 bg-red-900/20 px-2 py-1 rounded">
          {validation.dateRangeError}
        </div>
      )}
      
      {/* Cross-midnight indicator */}
      {isCrossMidnight && (
        <div className="text-xs text-purple-400 bg-purple-900/20 px-2 py-1 rounded">
          Cross-midnight: {validation.startParsed} → {validation.endParsed} (+1 day)
        </div>
      )}
    </div>
  );
}
