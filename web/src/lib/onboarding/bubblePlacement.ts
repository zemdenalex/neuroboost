export interface Rect { top: number; left: number; width: number; height: number }
export interface Size { width: number; height: number }
export interface Viewport { width: number; height: number }
export type Placement = 'top' | 'bottom' | 'left' | 'right'
export interface Placed { top: number; left: number; placement: Placement }

function clamp(value: number, max: number): number {
  // pins to 0 when max < 0 (bubble larger than viewport)
  return Math.max(0, Math.min(value, max))
}

export function placeBubble(anchor: Rect, bubble: Size, viewport: Viewport, gap = 8): Placed {
  const roomBelow = viewport.height - (anchor.top + anchor.height)
  const roomAbove = anchor.top
  const roomRight = viewport.width - (anchor.left + anchor.width)
  const roomLeft = anchor.left
  const needV = bubble.height + gap
  const needH = bubble.width + gap

  let placement: Placement
  if (roomBelow >= needV) placement = 'bottom'
  else if (roomAbove >= needV) placement = 'top'
  else if (roomRight >= needH) placement = 'right'
  else if (roomLeft >= needH) placement = 'left'
  else placement = 'bottom'

  const centerX = anchor.left + anchor.width / 2 - bubble.width / 2
  const centerY = anchor.top + anchor.height / 2 - bubble.height / 2

  let top: number
  let left: number
  switch (placement) {
    case 'bottom': top = anchor.top + anchor.height + gap; left = centerX; break
    case 'top': top = anchor.top - bubble.height - gap; left = centerX; break
    case 'right': left = anchor.left + anchor.width + gap; top = centerY; break
    case 'left': left = anchor.left - bubble.width - gap; top = centerY; break
  }

  return {
    placement,
    left: clamp(left, viewport.width - bubble.width),
    top: clamp(top, viewport.height - bubble.height),
  }
}
