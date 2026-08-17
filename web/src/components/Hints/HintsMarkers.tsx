import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Info } from 'lucide-react'
import { hintsForRoute } from '../../lib/onboarding/hintsContent'
import { placeBubble } from '../../lib/onboarding/bubblePlacement'

const DOT = 20
const BUBBLE = { width: 256, height: 100 }

interface Marker {
  anchor: string
  titleKey: string
  bodyKey: string
  dot: { top: number; left: number }
  rect: { top: number; left: number; width: number; height: number }
}

function clampDot(value: number, max: number): number {
  return Math.max(0, Math.min(value, max))
}

/**
 * The "markers" hint style: a small persistent ⓘ dot pinned near each present
 * `data-hint` anchor on the current page. Always rendered while this style is
 * selected (no trigger). Hovering (desktop) or tapping (touch) a dot reveals
 * that anchor's single bubble; clicking it again, opening another, or Escape
 * closes it. Non-blocking: the container lets clicks fall through; only the
 * dots and the open bubble capture pointer events.
 */
export function HintsMarkers() {
  const { pathname } = useLocation()
  const { t } = useTranslation('onboarding')
  const [markers, setMarkers] = useState<Marker[]>([])
  const [active, setActive] = useState<string | null>(null)
  const rafRef = useRef<number | null>(null)

  const measure = useCallback(() => {
    const vw = window.innerWidth
    const vh = window.innerHeight
    const next: Marker[] = []
    for (const h of hintsForRoute(pathname)) {
      const el = document.querySelector(`[data-hint="${h.anchor}"]`)
      if (!el) continue
      const r = el.getBoundingClientRect()
      if (r.width === 0 && r.height === 0) continue // hidden / not laid out
      next.push({
        anchor: h.anchor,
        titleKey: h.titleKey,
        bodyKey: h.bodyKey,
        dot: {
          top: clampDot(r.top - 8, vh - DOT),
          left: clampDot(r.left + r.width - 12, vw - DOT),
        },
        rect: { top: r.top, left: r.left, width: r.width, height: r.height },
      })
    }
    setMarkers(next)
  }, [pathname])

  useLayoutEffect(() => {
    measure()
  }, [measure])

  useEffect(() => {
    const onChange = () => {
      if (rafRef.current != null) return
      rafRef.current = requestAnimationFrame(() => {
        rafRef.current = null
        measure()
      })
    }
    window.addEventListener('resize', onChange)
    window.addEventListener('scroll', onChange, true)
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setActive(null)
    }
    document.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('resize', onChange)
      window.removeEventListener('scroll', onChange, true)
      document.removeEventListener('keydown', onKey)
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current)
    }
  }, [measure])

  // Close the open bubble if its anchor scrolled away / disappeared.
  useEffect(() => {
    if (active && !markers.some((m) => m.anchor === active)) setActive(null)
  }, [markers, active])

  if (markers.length === 0) return null

  const activeMarker = markers.find((m) => m.anchor === active) ?? null
  const bubblePos = activeMarker
    ? placeBubble(activeMarker.rect, BUBBLE, { width: window.innerWidth, height: window.innerHeight })
    : null

  return (
    <div className="pointer-events-none fixed inset-0 z-40">
      {markers.map((m) => (
        <button
          key={m.anchor}
          type="button"
          style={{ top: m.dot.top, left: m.dot.left, width: DOT, height: DOT }}
          onClick={() => setActive((a) => (a === m.anchor ? null : m.anchor))}
          onPointerEnter={(e) => {
            // Hover-to-preview on desktop only. On touch a tap also fires
            // pointerenter; guarding on 'mouse' lets the click toggle own touch.
            if (e.pointerType === 'mouse') setActive(m.anchor)
          }}
          onFocus={() => setActive(m.anchor)}
          aria-label={t('hints.reveal')}
          className="pointer-events-auto absolute flex items-center justify-center rounded-full border border-blue-400 bg-blue-600 text-white shadow-md transition-colors hover:bg-blue-500"
        >
          <Info className="h-3 w-3" />
        </button>
      ))}
      {activeMarker && bubblePos && (
        <div
          style={{ top: bubblePos.top, left: bubblePos.left, width: BUBBLE.width }}
          className="pointer-events-auto absolute rounded-lg border border-blue-500 bg-zinc-900 p-3 font-mono text-zinc-100 shadow-xl"
        >
          <p className="text-sm font-medium">{t(activeMarker.titleKey)}</p>
          <p className="mt-1 text-xs leading-relaxed text-zinc-400">{t(activeMarker.bodyKey)}</p>
        </div>
      )}
    </div>
  )
}

export default HintsMarkers
