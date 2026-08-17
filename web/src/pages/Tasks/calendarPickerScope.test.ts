import { describe, it, expect } from 'vitest'

/**
 * The task editor may offer a calendar only while CREATING.
 *
 * An existing task cannot be moved: UpdateTaskRequest carries no calendar_id
 * and the API would ignore one. Rendering the picker while editing would let
 * Denis change the calendar, press Save, get a 200, and watch nothing happen —
 * the silent-success failure that made him press an invitation button seven
 * times (learning-a-silent-success-reads-as-a-failure), except here the silence
 * is a discarded choice rather than a completed one.
 *
 * ⚠ DELETE THIS TEST when PATCH /api/tasks/:id learns calendar_id. At that
 * point the guard is wrong and this file is the thing standing in the way —
 * which is the point: it should be noticed, not quietly satisfied.
 *
 * Source-scanned because the page is a large hook-using component and this
 * project has no renderer in its test dependencies; adding one is not on the
 * table for a single assertion.
 */
describe('the task calendar picker is offered on create only', () => {
  const sources = import.meta.glob('./Tasks.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>

  const source = Object.values(sources)[0]

  it('found the page to check', () => {
    // The floor: a rename or a move would otherwise leave this scanning an
    // empty set and reporting success.
    expect(source, 'Tasks.tsx was not found — this test guards nothing').toBeTruthy()
    expect(source).toContain('CalendarPicker')
  })

  it('renders the picker behind a not-editing guard', () => {
    // Everything between the guard and the element, collapsed: the guard has to
    // be the thing that admits the picker, not merely present elsewhere.
    const guarded = /!editingTask\.id\s*&&\s*\(\s*<CalendarPicker/.test(
      source.replace(/\{\/\*[\s\S]*?\*\/\}/g, ''),
    )
    expect(
      guarded,
      'CalendarPicker is rendered without a !editingTask.id guard — editing would discard the choice',
    ).toBe(true)
  })

  it('omits calendar_id from the create payload rather than sending it empty', () => {
    // An empty string is not "no calendar" to the API's validator; the backend
    // treats an absent field as "my personal calendar" and that is the path
    // quick-add, the bot and the importer already take.
    expect(source).toContain("...(editingTask.calendar_id ? { calendar_id: editingTask.calendar_id } : {})")
  })
})
