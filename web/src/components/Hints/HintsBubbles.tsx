import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { hintsForRoute } from '../../lib/onboarding/hintsContent'
import { placeBubble, type Placed } from '../../lib/onboarding/bubblePlacement'
import { useOnboarding } from '../../contexts/OnboardingContext'

// Fixed estimate for placement; the bubble renders at this width and the
// placement is clamped to the viewport, so a slightly taller bubble still sits
// fully on-screen. Avoids a measure→render→re-measure loop.
const BUBBLE = { width: 256, height: 100 }

interface PositionedHint {
  anchor: string
  titleKey: string
  bodyKey: string
  pos: Placed
}

/**
 * The "bubbles" hint style: when active, anchors a labelled bubble to every
 * `data-hint` element resolved for the current route. Non-blocking — the layer
 * lets clicks fall through to the page; only the bubbles and the "Got it"
 * button capture pointer events. Dismissed via "Got it", Escape, or navigation
 * (the provider resets `hintsActive` on route change).
 */
export function HintsBubbles() {
  const { hideHints } = useOnboarding()
  const { pathname } = useLocation()
  const { t } = useTranslation('onboarding')
  const [positioned, setPositioned] = useState<PositionedHint[]>([])
  const gotItRef = useRef<HTMLButtonElement>(null)
  const rafRef = useRef<number | null>(null)

  const measure = useCallback(() => {
    const viewport = { width: window.innerWidth, height: window.innerHeight }
    const next: PositionedHint[] = []
    for (const h of hintsForRoute(pathname)) {
      const el = document.querySelector(`[data-hint="${h.anchor}"]`)
      if (!el) continue
      const r = el.getBoundingClientRect()
      if (r.width === 0 && r.height === 0) continue // hidden / not laid out
      next.push({
        anchor: h.anchor,
        titleKey: h.titleKey,
        bodyKey: h.bodyKey,
        pos: placeBubble({ top: r.top, left: r.left, width: r.width, height: r.height }, BUBBLE, viewport),
      })
    }
    setPositioned(next)
  }, [pathname])

  useLayoutEffect(() => {
    measure()
  }, [measure])

  useEffect(() => {
    // Coalesce bursts of scroll/resize events into one re-measure per frame.
    const onChange = () => {
      if (rafRef.current != null) return
      rafRef.current = requestAnimationFrame(() => {
        rafRef.current = null
        measure()
      })
    }
    window.addEventListener('resize', onChange)
    window.addEventListener('scroll', onChange, true) // capture: also catch scrolls inside inner containers
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') hideHints()
    }
    document.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('resize', onChange)
      window.removeEventListener('scroll', onChange, true)
      document.removeEventListener('keydown', onKey)
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current)
    }
  }, [measure, hideHints])

  useEffect(() => {
    if (positioned.length > 0) gotItRef.current?.focus()
  }, [positioned.length])

  if (positioned.length === 0) return null

  return (
    <div className="pointer-events-none fixed inset-0 z-50">
      {positioned.map((h) => (
        <div
          key={h.anchor}
          style={{ top: h.pos.top, left: h.pos.left, width: BUBBLE.width }}
          className="pointer-events-auto absolute rounded-lg border border-blue-500 bg-zinc-900 p-3 font-mono text-zinc-100 shadow-xl"
        >
          <p className="text-sm font-medium">{t(h.titleKey)}</p>
          <p className="mt-1 text-xs leading-relaxed text-zinc-400">{t(h.bodyKey)}</p>
        </div>
      ))}
      <div className="pointer-events-auto absolute bottom-6 left-1/2 -translate-x-1/2">
        <button
          ref={gotItRef}
          type="button"
          onClick={hideHints}
          className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-mono text-white shadow-lg transition-colors hover:bg-blue-700"
        >
          {t('hints.gotIt')}
        </button>
      </div>
    </div>
  )
}

export default HintsBubbles
