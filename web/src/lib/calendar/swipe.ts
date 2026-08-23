/**
 * Is this touch a sideways swipe between days, or an ordinary scroll?
 *
 * 🔴 The bug this exists to prevent (Denis, 23.08: "события исчезают при
 * прокрутке"). The grid recorded only `clientX` on touchstart, so on touchend
 * there was no vertical distance to compare against — `Math.abs(dx) > 50` was
 * the whole test. Scrolling a week down with a thumb drifts sideways by well
 * over 50px on a phone, so the day silently changed underneath the scroll.
 * Nothing disappeared: the user was looking at a different day.
 *
 * A pure function, because both halves of the fix are decisions and neither is
 * observable from the outside: which coordinates get recorded, and what counts
 * as horizontal. This is the second one.
 */

/** How far the finger must travel before it means anything. */
export const SWIPE_MIN_PX = 50

/**
 * How much more horizontal than vertical the movement has to be.
 *
 * 1.5 rather than 1: a 45° drag is ambiguous, and resolving ambiguity in favour
 * of "stay where you are" is the cheaper mistake — a swipe that does not fire
 * costs one more swipe, a page that jumps mid-scroll costs the reader their
 * place.
 */
export const SWIPE_DOMINANCE = 1.5

export function isHorizontalSwipe(dx: number, dy: number): boolean {
  return Math.abs(dx) > SWIPE_MIN_PX && Math.abs(dx) > Math.abs(dy) * SWIPE_DOMINANCE
}
