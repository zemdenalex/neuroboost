/**
 * Cross-component signal that the task list on the server has changed.
 *
 * The global Ctrl+K modal writes through the API while the Tasks page holds
 * its own copy of the list in state. Without a signal, a task captured from
 * the modal only appeared after a reload — which reads as "the shortcut did
 * nothing", the single most discouraging outcome for a capture flow.
 *
 * A DOM event rather than shared state: the modal is mounted in the layout,
 * far from the Tasks page, and the project already uses this pattern for
 * `neuroboost-layout-change`.
 */
export const TASKS_CHANGED_EVENT = 'neuroboost-tasks-changed'

export function announceTasksChanged(): void {
  window.dispatchEvent(new CustomEvent(TASKS_CHANGED_EVENT))
}

/** Returns an unsubscribe function, shaped for a useEffect cleanup. */
export function onTasksChanged(handler: () => void): () => void {
  window.addEventListener(TASKS_CHANGED_EVENT, handler)
  return () => window.removeEventListener(TASKS_CHANGED_EVENT, handler)
}
