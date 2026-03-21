import type { ReflectionState } from './editor.types';

interface ReflectionFieldsProps {
  reflection: ReflectionState;
  hasExistingReflection: boolean;
  onReflectionChange: (update: Partial<ReflectionState>) => void;
}

export function ReflectionFields({
  reflection,
  hasExistingReflection,
  onReflectionChange,
}: ReflectionFieldsProps) {
  return (
    <div className="space-y-3 p-3 bg-zinc-800 rounded border border-zinc-600">
      {hasExistingReflection && (
        <div className="text-xs text-green-400 mb-2">
          ✓ Reflection saved
        </div>
      )}
      
      <div className="grid grid-cols-3 gap-3">
        <SliderField
          label="Focus"
          value={reflection.focus}
          min={1}
          max={10}
          step={1}
          displayValue={`${reflection.focus}/10`}
          onChange={(v) => onReflectionChange({ focus: v })}
        />
        <SliderField
          label="Energy"
          value={reflection.energy}
          min={1}
          max={10}
          step={1}
          displayValue={`${reflection.energy}/10`}
          onChange={(v) => onReflectionChange({ energy: v })}
        />
        <SliderField
          label="Mood"
          value={reflection.mood}
          min={1}
          max={10}
          step={1}
          displayValue={`${reflection.mood}/10`}
          onChange={(v) => onReflectionChange({ mood: v })}
        />
      </div>
      
      <textarea
        placeholder="Reflection notes (optional)..."
        value={reflection.note}
        onChange={(e) => onReflectionChange({ note: e.target.value })}
        rows={2}
        className="w-full px-2 py-1 bg-zinc-700 border border-zinc-600 rounded text-white placeholder-zinc-400 focus:outline-none focus:border-zinc-400 resize-none text-sm font-mono"
      />
      
      <div className="text-xs text-zinc-500">
        Track how the event went: focus level, goal completion, and mood
      </div>
    </div>
  );
}

interface SliderFieldProps {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  displayValue: string;
  onChange: (value: number) => void;
}

function SliderField({ label, value, min, max, step, displayValue, onChange }: SliderFieldProps) {
  return (
    <div>
      <label className="block text-xs text-zinc-400 mb-1">{label}</label>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full h-2 bg-zinc-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
      />
      <div className="text-xs text-center mt-1 text-zinc-300 font-mono">
        {displayValue}
      </div>
    </div>
  );
}

/** Get color based on value for visual feedback */
export function getValueColor(value: number, max: number): string {
  const ratio = value / max;
  if (ratio >= 0.8) return 'text-green-400';
  if (ratio >= 0.5) return 'text-yellow-400';
  return 'text-red-400';
}
