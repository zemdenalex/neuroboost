// WeekGrid layout constants
export const HOUR_PX = 44;
export const MIN_SLOT_MIN = 15;
export const ALL_DAY_HEIGHT = 80;
export const DAY_HEADER_HEIGHT = 28;
export const DAY_MS = 24 * 60 * 60 * 1000;

// Auto-scroll settings
export const EDGE_THRESHOLD = 24;
export const SCROLL_SPEED = 8;

// Mobile breakpoints for responsive day count
export const MOBILE_BREAKPOINT = 480;
export const TABLET_BREAKPOINT = 768;

// Priority colors for task integration
export const PRIORITY_COLORS: Record<number, string> = {
  0: 'bg-zinc-600', // Buffer
  1: 'bg-red-600',  // Emergency
  2: 'bg-orange-600', // ASAP
  3: 'bg-yellow-600', // Must today
  4: 'bg-green-600', // Deadline soon
  5: 'bg-blue-600', // If possible
};

// Ghost preview colors
export const GHOST_COLORS = {
  create: 'bg-emerald-400/40 border-emerald-400/60',
  move: 'bg-blue-400/40 border-blue-400/60',
  resize: 'bg-amber-400/40 border-amber-400/60',
  multiDay: 'bg-purple-400/40 border-purple-400/60',
} as const;
