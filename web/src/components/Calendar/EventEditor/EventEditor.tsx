import { useTranslation } from 'react-i18next';
import { DateTimeFields } from './DateTimeFields';
import { BasicFields } from './BasicFields';
import { AdvancedFields } from './AdvancedFields';
import { RepeatFields } from './RepeatFields';
import { ReflectionFields } from './ReflectionFields';
import { useEditorForm } from './useEditorForm';
import type { EditorProps } from './editor.types';

export function EventEditor({ 
  range, 
  draft, 
  timezone, 
  onClose, 
  onCreated, 
  onPatched, 
  onDelete 
}: EditorProps) {
  const { t } = useTranslation('calendar');
  const { t: tc } = useTranslation('common');
  const { state, actions, isEditing, hasReflection, canSave } = useEditorForm(
    draft, range, timezone, onCreated, onPatched, onDelete
  );

  return (
    <div 
      className="bg-zinc-900 border border-zinc-700 rounded-lg p-6 w-full max-w-md max-h-[90vh] overflow-y-auto"
      onClick={(e) => e.stopPropagation()}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-white">
          {isEditing ? t('editEvent') : t('newEvent')}
        </h3>
        <button 
          onClick={onClose} 
          className="text-zinc-400 hover:text-zinc-200 text-xl leading-none"
        >
          ×
        </button>
      </div>
      
      <div className="space-y-4">
        {/* Date/Time fields (hidden for all-day events) */}
        {!state.isAllDay && (
          <DateTimeFields
            startDate={state.startDateLocal}
            endDate={state.endDateLocal}
            startTime={state.startTimeInput}
            endTime={state.endTimeInput}
            validation={state.validation}
            timezone={timezone}
            onStartDateChange={actions.setStartDateLocal}
            onEndDateChange={actions.setEndDateLocal}
            onStartTimeChange={(v, p) => actions.handleTimeChange(v, p, true)}
            onEndTimeChange={(v, p) => actions.handleTimeChange(v, p, false)}
            onEndTimeEnter={actions.handleSave}
          />
        )}

        {/* Basic fields (title always, description/location/tags when advanced) */}
        <BasicFields
          title={state.title}
          description={state.description}
          location={state.location}
          tags={state.tags}
          showAdvanced={state.showAdvanced}
          onTitleChange={actions.setTitle}
          onDescriptionChange={actions.setDescription}
          onLocationChange={actions.setLocation}
          onTagsChange={actions.setTags}
          onTitleEnter={actions.handleSave}
          onDescriptionCtrlEnter={actions.handleSave}
        />

        {/* Advanced fields (all-day, reminder, color, repeat) */}
        {state.showAdvanced && (
          <>
            <AdvancedFields
              isAllDay={state.isAllDay}
              reminderMinutes={state.reminderMinutes}
              color={state.color}
              onAllDayChange={actions.setIsAllDay}
              onReminderChange={actions.setReminderMinutes}
              onColorChange={actions.setColor}
            />
            <RepeatFields
              repeatType={state.repeatType}
              repeatEndType={state.repeatEndType}
              repeatCount={state.repeatCount}
              repeatUntil={state.repeatUntil}
              onRepeatTypeChange={actions.setRepeatType}
              onRepeatEndTypeChange={actions.setRepeatEndType}
              onRepeatCountChange={actions.setRepeatCount}
              onRepeatUntilChange={actions.setRepeatUntil}
            />
          </>
        )}

        {/* Reflection section (only when editing) */}
        {isEditing && (
          <div>
            <button
              onClick={() => actions.setShowReflection(!state.showReflection)}
              className="text-sm text-zinc-400 hover:text-zinc-200 mb-2"
            >
              {hasReflection ? '✓ ' : ''}{t('reflection')} {state.showReflection ? '−' : '+'}
            </button>
            
            {state.showReflection && (
              <ReflectionFields
                reflection={state.reflection}
                hasExistingReflection={hasReflection}
                onReflectionChange={actions.handleReflectionChange}
              />
            )}
          </div>
        )}
      </div>

      {/* Footer actions */}
      <div className="flex items-center justify-between mt-6 pt-4 border-t border-zinc-700">
        <div className="flex gap-2">
          <button
            onClick={() => actions.setShowAdvanced(!state.showAdvanced)}
            className="text-sm text-zinc-400 hover:text-zinc-200"
          >
            {state.showAdvanced ? t('simple') : t('advanced')} {state.showAdvanced ? '−' : '+'}
          </button>
          
          {isEditing && (
            <button
              onClick={actions.handleDelete}
              disabled={state.isDeleting}
              className="text-sm text-red-400 hover:text-red-300 disabled:text-red-600"
            >
              {state.isDeleting ? t('deleting') : t('delete')}
            </button>
          )}
        </div>

        <div className="flex gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 text-zinc-400 hover:text-zinc-200 text-sm"
          >
            {tc('action.cancel')}
          </button>
          <button
            onClick={actions.handleSave}
            disabled={!canSave}
            className="px-4 py-2 bg-zinc-700 hover:bg-zinc-600 disabled:bg-zinc-800 disabled:text-zinc-500 text-white rounded text-sm"
          >
            {isEditing ? (state.showReflection ? t('saveReflect') : t('save')) : t('create')}
          </button>
        </div>
      </div>
    </div>
  );
}
