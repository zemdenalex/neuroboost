// WeekGrid layout constants
export const HOUR_PX = 44;
export const MIN_SLOT_MIN = 15;
export const ALL_DAY_HEIGHT = 80;
export const DAY_HEADER_HEIGHT = 28;
export const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * Pointer travel before a mousedown counts as a drag rather than a click.
 * Applies to move and to both resize directions.
 */
export const DRAG_THRESHOLD_PX = 5;

/**
 * Touch: how long a finger must stay on an event before it is picked up, and how
 * far it may drift meanwhile. Past the slop it is a scroll, not a hold. Our own
 * choice (Apple and Google publish no number); to become a setting.
 */
export const TOUCH_HOLD_MS = 400;
export const TOUCH_SLOP_PX = 8;

// Auto-scroll settings
export const EDGE_THRESHOLD = 24;

/**
 * Phone: a held event parked this close to the left or right edge of the grid
 * flips the day after EDGE_FLIP_MS, and keeps flipping while it stays there
 * (step 9 of docs/tasks-phone-drag.md: «через ~0,8 с день листается»).
 */
export const EDGE_FLIP_PX = 32;
export const EDGE_FLIP_MS = 800;
export const SCROLL_SPEED = 8;

// Mobile breakpoints for responsive day count — lib/calendar/visibleDays.
export { MOBILE_BREAKPOINT, TABLET_BREAKPOINT } from '../../../lib/calendar/visibleDays';

// Priority colors for task integration — from lib/priority (single source of truth).
export { PRIORITY_BLOCK_COLORS as PRIORITY_COLORS } from '../../../lib/priority';

// Ghost preview colors
export const GHOST_COLORS = {
  create: 'bg-emerald-400/40 border-emerald-400/60',
  move: 'bg-blue-400/40 border-blue-400/60',
  resize: 'bg-amber-400/40 border-amber-400/60',
  multiDay: 'bg-purple-400/40 border-purple-400/60',
} as const;
