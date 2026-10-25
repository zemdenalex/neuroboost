/**
 * How a task row offers its actions on a phone (Denis 25.09, variants page
 * https://claude.ai/artifact/GqNKUBaqERKbD6vFzJRyBD: «all three, customizable
 * in settings»). Three icons always visible took ~110px of a 375px row, so
 * titles wrapped and the trash sat next to the pencil.
 *
 * - `menu`: one «⋯» button, a menu with Schedule / Edit / Delete (default);
 * - `swipe`: swipe the row left to reveal Schedule and Delete, tap to edit;
 * - `card`: tap the row, a sheet with the actions slides up.
 *
 * A top-level account setting like month_view_variant: the settings saver
 * merges shallowly. The desktop keeps its hover icons whatever is chosen.
 */

export const ROW_ACTIONS = ['menu', 'swipe', 'card'] as const
export type RowActions = (typeof ROW_ACTIONS)[number]

export function readRowActions(settings: { task_row_actions?: unknown } | undefined): RowActions {
  const v = settings?.task_row_actions
  return typeof v === 'string' && (ROW_ACTIONS as readonly string[]).includes(v) ? (v as RowActions) : 'menu'
}

/** Width of the two revealed buttons (Schedule, Delete). */
export const SWIPE_ACTIONS_PX = 128

/** Row offset for a finger that moved dx px; left only, capped at the actions. */
export function swipeOffset(dx: number, wasOpen: boolean): number {
  const raw = (wasOpen ? -SWIPE_ACTIONS_PX : 0) + dx
  return Math.max(-SWIPE_ACTIONS_PX, Math.min(0, raw))
}

/** Whether a released row stays open: past a third of the actions width. */
export function swipeSettle(offset: number): boolean {
  return offset < -SWIPE_ACTIONS_PX / 3
}
