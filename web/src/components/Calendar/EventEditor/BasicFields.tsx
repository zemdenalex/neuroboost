import { useRef, useEffect } from 'react';

interface BasicFieldsProps {
  title: string;
  description: string;
  location: string;
  tags: string;
  showAdvanced: boolean;
  onTitleChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onLocationChange: (value: string) => void;
  onTagsChange: (value: string) => void;
  onTitleEnter?: () => void;
  onDescriptionCtrlEnter?: () => void;
  autoFocusTitle?: boolean;
}

export function BasicFields({
  title,
  description,
  location,
  tags,
  showAdvanced,
  onTitleChange,
  onDescriptionChange,
  onLocationChange,
  onTagsChange,
  onTitleEnter,
  onDescriptionCtrlEnter,
  autoFocusTitle = true,
}: BasicFieldsProps) {
  const titleRef = useRef<HTMLInputElement>(null);
  const descriptionRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (autoFocusTitle) {
      titleRef.current?.focus();
    }
  }, [autoFocusTitle]);

  const handleTitleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      onTitleEnter?.();
    }
  };

  const handleDescriptionKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && e.ctrlKey) {
      e.preventDefault();
      onDescriptionCtrlEnter?.();
    }
  };

  return (
    <>
      {/* Title - always visible */}
      <div>
        <input
          ref={titleRef}
          type="text"
          placeholder="Event title..."
          value={title}
          onChange={(e) => onTitleChange(e.target.value)}
          onKeyDown={handleTitleKeyDown}
          className="w-full px-3 py-2 bg-zinc-800 border border-zinc-600 rounded text-white placeholder-zinc-400 focus:outline-none focus:border-zinc-400 font-mono"
        />
      </div>

      {/* Advanced fields */}
      {showAdvanced && (
        <>
          <div>
            <textarea
              ref={descriptionRef}
              placeholder="Description (optional)..."
              value={description}
              onChange={(e) => onDescriptionChange(e.target.value)}
              onKeyDown={handleDescriptionKeyDown}
              rows={3}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-600 rounded text-white placeholder-zinc-400 focus:outline-none focus:border-zinc-400 resize-none font-mono"
            />
            <div className="text-xs text-zinc-500 mt-1">
              Ctrl+Enter to save
            </div>
          </div>

          <div>
            <input
              type="text"
              placeholder="Location (optional)..."
              value={location}
              onChange={(e) => onLocationChange(e.target.value)}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-600 rounded text-white placeholder-zinc-400 focus:outline-none focus:border-zinc-400 font-mono"
            />
          </div>

          <div>
            <input
              type="text"
              placeholder="Tags (comma separated)..."
              value={tags}
              onChange={(e) => onTagsChange(e.target.value)}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-600 rounded text-white placeholder-zinc-400 focus:outline-none focus:border-zinc-400 font-mono"
            />
          </div>
        </>
      )}
    </>
  );
}

export function getTitleRef() {
  return useRef<HTMLInputElement>(null);
}

export function getDescriptionRef() {
  return useRef<HTMLTextAreaElement>(null);
}
