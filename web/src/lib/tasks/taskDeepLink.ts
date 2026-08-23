/**
 * Opening one task from somewhere else in the app.
 *
 * 🔴 Why this exists. Tapping a task in the calendar did nothing at all:
 * `onSelectTask` and `onEditTask` were `console.log` stubs, wired into both the
 * desktop sidebar and the mobile panel. They were the only two stubs left in
 * `src/pages/`, and from the outside a stub is indistinguishable from a slow
 * network — you tap again.
 *
 * The tasks page already knows how to show and edit a task, so the calendar
 * sends the user there rather than growing a second editor. This is the link
 * format both sides agree on, kept apart from either so the agreement can be
 * asserted without rendering a page.
 */

export interface TaskDeepLink {
  taskId: string
  /** true → open the editor; false → show it in the list and highlight it. */
  edit: boolean
}

/** Build the URL the calendar navigates to. */
export function taskDeepLinkTo(taskId: string, edit: boolean): string {
  return `/tasks?task=${encodeURIComponent(taskId)}${edit ? '&edit=1' : ''}`
}

/** Read it back on the tasks page. Returns null when there is nothing to open. */
export function parseTaskDeepLink(params: URLSearchParams): TaskDeepLink | null {
  const taskId = params.get('task')
  if (!taskId) return null
  // Anything other than the exact flag is "just show me the task": a link that
  // has been edited by hand should not silently open an editor.
  return { taskId, edit: params.get('edit') === '1' }
}
