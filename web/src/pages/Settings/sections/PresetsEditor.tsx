import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { ReminderOffsets } from '../../../components/ReminderOffsets/ReminderOffsets'
import type { ReminderSettings } from '../../../lib/reminders/offsets'
import { presetLabel } from '../../../lib/reminders/presetLabel'
import {
  addPreset,
  deletePreset,
  duplicateOffsetGroups,
  renamePreset,
  type PresetError,
} from '../../../lib/reminders/presets'

interface Props {
  settings: ReminderSettings
  onChange: (next: ReminderSettings) => void
}

/**
 * The preset list: rename, delete, add, and edit each preset's offsets.
 *
 * Split out of RemindersSection rather than grown inside it — the section was
 * already the largest of the thirteen, and this adds draft state that has
 * nothing to do with the digest or the quiet hours beside it.
 *
 * Two shapes here are deliberate and easy to "simplify" back into defects:
 *
 * 1. A name commits on blur or Enter, from a draft held here — not on every
 *    keystroke like every other control in Settings. Saving per keystroke
 *    would rename `обычное` to `о`, `об`, `обы`… leaving a trail of presets,
 *    repointing the defaults at each one, and racing the `user` refresh that
 *    follows each save and would overwrite the half-typed field.
 * 2. Only one row is editable at a time (`editing`), so the draft cannot be
 *    orphaned by a rename happening in a different row.
 */
export function PresetsEditor({ settings, onChange }: Props) {
  const { t: tr } = useTranslation('reminders')
  const [editing, setEditing] = useState<{ name: string; draft: string } | null>(null)
  const [newName, setNewName] = useState('')
  // Keyed by the row the message belongs to; '' is the add row.
  const [errors, setErrors] = useState<Record<string, PresetError>>({})

  const setError = (row: string, reason: PresetError | null) =>
    setErrors(prev => {
      const next = { ...prev }
      if (reason) next[row] = reason
      else delete next[row]
      return next
    })

  const commitRename = (from: string) => {
    if (!editing || editing.name !== from) return
    const result = renamePreset(settings, from, editing.draft)
    if (!result.ok) {
      setError(from, result.reason)
      return
    }
    setError(from, null)
    setEditing(null)
    onChange(result.settings)
  }

  const commitAdd = () => {
    const result = addPreset(settings, newName)
    if (!result.ok) {
      setError('', result.reason)
      return
    }
    setError('', null)
    setNewName('')
    onChange(result.settings)
  }

  const remove = (name: string) => {
    const result = deletePreset(settings, name)
    if (!result.ok) {
      setError(name, result.reason)
      return
    }
    if (editing?.name === name) setEditing(null)
    setError(name, null)
    onChange(result.settings)
  }

  const duplicates = duplicateOffsetGroups(settings.presets)

  return (
    <div>
      <h3 className="text-sm text-zinc-400 mb-1">{tr('settings.presetsTitle')}</h3>
      <p className="mb-2 text-xs text-zinc-500">{tr('settings.presetsHint')}</p>

      {duplicates.map(group => (
        <p key={group.join(',')} className="mb-2 text-xs text-amber-400">
          {/* Through presetLabel: the warning names presets, and the two
              dropdowns 30px above show those same presets by their TRANSLATED
              names. Printing the stored ones would tell an English reader that
              "важное, обычное" collide when the screen only ever says
              "Important" and "Normal". */}
          {tr('settings.presetDuplicateWarning', {
            names: group.map(name => presetLabel(name, tr)).join(', '),
          })}
        </p>
      ))}

      <div className="space-y-3">
        {Object.entries(settings.presets).map(([name, offsets]) => (
          <div key={name} className="rounded-lg border border-zinc-800 p-3">
            {/* flex-wrap, not a fixed row: at 375px the name field and the
                delete button do not fit side by side, and the last thing this
                editor should do is push its own delete button off-screen. */}
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <input
                aria-label={tr('settings.presetNameLabel')}
                value={editing?.name === name ? editing.draft : name}
                onFocus={() => setEditing({ name, draft: name })}
                onChange={e => setEditing({ name, draft: e.target.value })}
                onBlur={() => commitRename(name)}
                onKeyDown={e => {
                  if (e.key === 'Enter') e.currentTarget.blur()
                  if (e.key === 'Escape') {
                    setEditing(null)
                    setError(name, null)
                  }
                }}
                className="min-w-0 flex-1 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 font-mono text-sm text-white focus:border-blue-500 focus:outline-none"
              />
              <button
                type="button"
                aria-label={tr('settings.presetDelete')}
                title={tr('settings.presetDelete')}
                onClick={() => remove(name)}
                className="rounded-lg border border-zinc-700 p-2 text-zinc-400 hover:border-red-500 hover:text-red-400"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
            {errors[name] && (
              <p className="mb-2 text-xs text-red-400">{tr(`settings.presetError.${errors[name]}`)}</p>
            )}
            <ReminderOffsets
              value={offsets}
              presets={settings.presets}
              // 🔴 This row EDITS the preset named above it. A preset chooser
              // inside it overwrites that preset with another one's offsets —
              // which is how all three collapsed to the same values on staging.
              showPresetPicker={false}
              onChange={next => onChange({ ...settings, presets: { ...settings.presets, [name]: next } })}
            />
          </div>
        ))}
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <input
          value={newName}
          placeholder={tr('settings.presetNamePlaceholder')}
          onChange={e => setNewName(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              e.preventDefault()
              commitAdd()
            }
          }}
          className="min-w-0 flex-1 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 font-mono text-sm text-white focus:border-blue-500 focus:outline-none"
        />
        <button
          type="button"
          onClick={commitAdd}
          className="flex items-center gap-1 rounded-lg border border-zinc-700 px-3 py-2 text-sm text-zinc-300 hover:border-blue-500 hover:text-white"
        >
          <Plus className="h-4 w-4" />
          {tr('settings.presetAdd')}
        </button>
      </div>
      {errors[''] && <p className="mt-2 text-xs text-red-400">{tr(`settings.presetError.${errors['']}`)}</p>}
    </div>
  )
}
