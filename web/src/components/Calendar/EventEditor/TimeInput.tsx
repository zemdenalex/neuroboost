import { useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { parseTimeInput } from './editor.utils';

interface TimeInputProps {
  value: string;
  onChange: (value: string, parsed: string | null) => void;
  onBlur?: () => void;
  onEnter?: () => void;
  placeholder?: string;
  label?: string;
  error?: string;
  autoFocus?: boolean;
  timezone?: string;
}

export function TimeInput({
  value,
  onChange,
  onBlur,
  onEnter,
  placeholder = 'e.g. 1050, 10:50',
  label,
  error,
  autoFocus,
  timezone,
}: TimeInputProps) {
  const { t } = useTranslation('calendar');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (autoFocus) {
      inputRef.current?.focus();
    }
  }, [autoFocus]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    const parsed = parseTimeInput(newValue);
    onChange(newValue, parsed);
  };

  const handleBlur = () => {
    // Auto-format on blur: "1050" becomes "10:50"
    const parsed = parseTimeInput(value);
    if (parsed && parsed !== value) {
      onChange(parsed, parsed);
    }
    onBlur?.();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      onEnter?.();
    }
    if (e.key === 'Escape') {
      inputRef.current?.blur();
    }
  };

  const isValid = !value || parseTimeInput(value) !== null;

  return (
    <div>
      {label && (
        <label className="block text-xs text-zinc-400 mb-1">
          {label}
          {timezone && <span className="text-zinc-500 ml-1">({timezone})</span>}
        </label>
      )}
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={handleChange}
        onBlur={handleBlur}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className={`w-full px-2 py-1 bg-zinc-800 border rounded text-white text-sm 
          focus:outline-none focus:border-zinc-400 font-mono
          ${isValid ? 'border-zinc-600' : 'border-red-500'}`}
      />
      {!isValid && value && (
        <div className="text-xs text-red-400 mt-1">
          {error || t('invalidTimeFormat')}
        </div>
      )}
    </div>
  );
}
