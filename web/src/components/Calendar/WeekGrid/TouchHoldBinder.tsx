import { memo, useEffect, useRef } from 'react';
import { createHoldTracker } from '../../../lib/calendar/touchHold';
import { TOUCH_HOLD_MS, TOUCH_SLOP_PX } from './weekgrid.constants';

interface Props {
  /**
   * A finger held on the parent block until it was picked up (TOUCH_HOLD_MS,
   * still). Receives the block's element and the finger's position.
   */
  onLift: (el: HTMLElement, clientX: number, clientY: number) => void;
}

/**
 * Gives the block it sits in the touch behaviour of «hold to pick up».
 *
 * A component of its own, rendered as a child, because EventBlock is called as a
 * plain function by its tests (eventPainting, sharedBadge: «memo()-wrapped and
 * hookless») and hooks cannot live there. The binder finds its block through its
 * own placeholder's parent.
 *
 * Native listeners, not React props: React attaches touch listeners as passive,
 * and a passive touchmove cannot stop the page from scrolling. This one is
 * non-passive and present from the first touchstart, the only way the browser
 * agrees to wait for it. Cost: a scroll that STARTS on an event waits for this
 * handler; a scroll that starts on an empty slot is unaffected.
 */
export const TouchHoldBinder = memo(function TouchHoldBinder({ onLift }: Props) {
  const marker = useRef<HTMLSpanElement>(null);
  const liftRef = useRef(onLift);
  liftRef.current = onLift;

  useEffect(() => {
    const el = marker.current?.parentElement;
    if (!el) return;
    const tracker = createHoldTracker({
      holdMs: TOUCH_HOLD_MS,
      slopPx: TOUCH_SLOP_PX,
      onLift: (x, y) => {
        // A short buzz where the platform has one; the finger learns it was picked up.
        try { navigator.vibrate?.(12); } catch { /* not allowed or unsupported */ }
        liftRef.current(el, x, y);
      },
    });
    const onStart = (e: TouchEvent) => {
      // One finger only: a second one is a pinch or a palm, not a pick-up.
      if (e.touches.length !== 1) { tracker.end(); return; }
      tracker.start(e.touches[0].clientX, e.touches[0].clientY);
    };
    const onMove = (e: TouchEvent) => {
      const t = e.touches[0];
      if (!t) return;
      tracker.move(t.clientX, t.clientY);
      // Picked up: the finger now moves the event, not the page.
      if (tracker.lifted() && e.cancelable) e.preventDefault();
    };
    const onEnd = () => tracker.end();
    // A long press raises the browser's context menu on a phone; while a hold is
    // running or done it would sit on top of the event being dragged.
    const onContextMenu = (e: Event) => { if (tracker.active()) e.preventDefault(); };
    el.addEventListener('touchstart', onStart, { passive: true });
    el.addEventListener('touchmove', onMove, { passive: false });
    el.addEventListener('touchend', onEnd);
    el.addEventListener('touchcancel', onEnd);
    el.addEventListener('contextmenu', onContextMenu);
    return () => {
      tracker.end();
      el.removeEventListener('touchstart', onStart);
      el.removeEventListener('touchmove', onMove);
      el.removeEventListener('touchend', onEnd);
      el.removeEventListener('touchcancel', onEnd);
      el.removeEventListener('contextmenu', onContextMenu);
    };
  }, []);

  return <span ref={marker} hidden aria-hidden="true" />;
});
