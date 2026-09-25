import type { ReflectionBody } from './editor.types'

/**
 * The reflection the event editor saves with an event.
 *
 * 🔴 Only what the editor shows: sliders and a note. The completed / on-time
 * flags live on the Reflections page; sending them from here (as true, until
 * 25.09) overwrote whatever was set there on every event save.
 */
export function editorReflectionBody(r: { focus: number; energy: number; mood: number; note: string }): ReflectionBody {
  return { focus: r.focus, energy: r.energy, mood: r.mood, note: r.note.trim() || undefined }
}
