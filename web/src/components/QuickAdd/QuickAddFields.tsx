import { useTranslation } from 'react-i18next'
import type { CreateTaskRequest } from '../../api/tasks'
import { PRIORITY_LABELS } from '../../lib/priority'
import { toDateTimeLocalValue, fromDateTimeLocalValue } from '../../lib/datetime/dateTimeLocal'
import { ReminderOffsets } from '../ReminderOffsets/ReminderOffsets'
import { useReminderSettings } from '../../hooks/useReminderSettings'

interface QuickAddFieldsProps {
  level: 1 | 2
  draft: Partial<CreateTaskRequest>
  onChange: (patch: Partial<CreateTaskRequest>) => void
}

const FIELD_CLASS =
  'w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500'

/**
 * The fields the quick-add row reveals when expanded.
 *
 * Level 1 is the context a task usually needs; level 2 is everything else.
 * Enter is not handled here on purpose — it keeps its native meaning in every
 * one of these controls, and only Ctrl+Enter (handled by the row) submits.
 */
export function QuickAddFields({ level, draft, onChange }: QuickAddFieldsProps) {
  const reminderSettings = useReminderSettings()
  const { t } = useTranslation('tasks')

  return (
    <div className="grid gap-3 rounded-lg border border-zinc-800 bg-zinc-900/60 p-3 sm:grid-cols-2">
      <div>
        <label className="mb-1 block text-sm text-zinc-400" htmlFor="qa-priority">{t('form.priority')}</label>
        <select
          id="qa-priority"
          value={draft.priority ?? 3}
          onChange={e => onChange({ priority: Number(e.target.value) })}
          className={FIELD_CLASS}
        >
          {Object.keys(PRIORITY_LABELS).map(value => (
            <option key={value} value={value}>{t(`priority.${value}`)}</option>
          ))}
        </select>
      </div>

      <div>
        <label className="mb-1 block text-sm text-zinc-400" htmlFor="qa-estimate">{t('form.estimatedTime')}</label>
        <input
          id="qa-estimate"
          type="number"
          min={1}
          value={draft.estimated_minutes ?? ''}
          onChange={e => onChange({ estimated_minutes: Number(e.target.value) || undefined })}
          placeholder={t('form.estimatedPlaceholder')}
          className={FIELD_CLASS}
        />
      </div>

      <div>
        <label className="mb-1 block text-sm text-zinc-400" htmlFor="qa-due">{t('form.dueDate')}</label>
        <input
          id="qa-due"
          type="datetime-local"
          value={draft.due_date ? toDateTimeLocalValue(draft.due_date) : ''}
          onChange={e => onChange({ due_date: e.target.value ? fromDateTimeLocalValue(e.target.value) : undefined })}
          className={FIELD_CLASS}
        />
      </div>

      <div>
        <label className="mb-1 block text-sm text-zinc-400" htmlFor="qa-tags">{t('quickAdd.tags')}</label>
        <input
          id="qa-tags"
          value={(draft.tags ?? []).join(', ')}
          onChange={e => onChange({ tags: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })}
          placeholder={t('quickAdd.tagsPlaceholder')}
          className={FIELD_CLASS}
        />
      </div>

      {level === 2 && (
        <div className="sm:col-span-2">
          {/* Reminders count back from due_date, so without one there is
              nothing to count from — the control says so rather than
              silently accepting offsets that could never fire. */}
          <ReminderOffsets
            value={draft.reminder_offsets ?? []}
            onChange={offsets => onChange({ reminder_offsets: offsets })}
            presets={reminderSettings.presets}
            disabled={!draft.due_date}
          />
        </div>
      )}

      {level === 2 && (
        <div className="sm:col-span-2">
          <label className="mb-1 block text-sm text-zinc-400" htmlFor="qa-description">{t('form.description')}</label>
          <textarea
            id="qa-description"
            rows={3}
            value={draft.description ?? ''}
            onChange={e => onChange({ description: e.target.value })}
            placeholder={t('form.descriptionPlaceholder')}
            className={FIELD_CLASS}
          />
        </div>
      )}
    </div>
  )
}
