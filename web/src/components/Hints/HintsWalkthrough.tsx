import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { hintsForRoute, type HintAnchor } from '../../lib/onboarding/hintsContent'
import { placeBubble, type Placed } from '../../lib/onboarding/bubblePlacement'
import { clampStep, nextStep } from '../../lib/onboarding/walkthroughStep'
import { useOnboarding } from '../../contexts/OnboardingContext'

const BUBBLE = { width: 256, height: 120 }

interface Measured {
  rect: { top: number; left: number; width: number; height: number }
  pos: Placed
}

/**
 * The "walkthrough" hint style: a guided, one-at-a-time tour of the current
 * page's `data-hint` anchors. Dims the page with a box-shadow "spotlight" around
 * the current anchor, scrolls it into view, and steps with Back/Next driven by
 * the pure walkthroughStep core. Modal: background clicks are inert; Escape or
 * finishing closes. Absent / zero-size anchors are excluded from the tour.
 */
export function HintsWalkthrough() {
  const { hideHints } = useOnboarding()
  const { pathname } = useLocation()
  const { t } = useTranslation('onboarding')
  const [steps, setSteps] = useState<HintAnchor[]>([])
  const [step, setStep] = useState(0)
  const [measured, setMeasured] = useState<Measured | null>(null)
  const rafRef = useRef<number | null>(null)
  const nextRef = useRef<HTMLButtonElement>(null)

  // Build the list of anchors actually present on this route, once per route.
  useLayoutEffect(() => {
    const present = hintsForRoute(pathname).filter((h) => {
      const el = document.querySelector(`[data-hint="${h.anchor}"]`)
      if (!el) return false
      const r = el.getBoundingClientRect()
      return !(r.width === 0 && r.height === 0)
    })
    setSteps(present)
    setStep(0)
  }, [pathname])

  const state = clampStep(step, steps.length)
  const current = steps[state.index]

  const measure = useCallback(() => {
    if (!current) {
      setMeasured(null)
      return
    }
    const el = document.querySelector(`[data-hint="${current.anchor}"]`)
    if (!el) {
      setMeasured(null)
      return
    }
    const r = el.getBoundingClientRect()
    const viewport = { width: window.innerWidth, height: window.innerHeight }
    const rect = { top: r.top, left: r.left, width: r.width, height: r.height }
    setMeasured({ rect, pos: placeBubble(rect, BUBBLE, viewport) })
  }, [current])

  // Scroll the current anchor into view, then measure.
  useLayoutEffect(() => {
    if (!current) return
    const el = document.querySelector(`[data-hint="${current.anchor}"]`)
    el?.scrollIntoView({ block: 'center', behavior: 'smooth' })
    measure()
  }, [current, measure])

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

  // Focus the primary button when the step changes (not on every re-measure —
  // `current` is stable across scroll/resize, so this won't steal focus mid-scroll).
  useEffect(() => {
    nextRef.current?.focus()
  }, [current])

  if (steps.length === 0 || !measured) return null

  const handleNext = () => {
    if (state.isLast) {
      hideHints()
      return
    }
    setStep((s) => nextStep(s, steps.length, 1).index)
  }
  const handleBack = () => setStep((s) => nextStep(s, steps.length, -1).index)

  return (
    <div className="fixed inset-0 z-50" role="dialog" aria-modal="true" aria-label={t(current.titleKey)}>
      {/* Spotlight: a ring around the anchor with a huge box-shadow dimming the rest. */}
      <div
        style={{
          top: measured.rect.top,
          left: measured.rect.left,
          width: measured.rect.width,
          height: measured.rect.height,
          boxShadow: '0 0 0 9999px rgba(0,0,0,0.65)',
        }}
        className="pointer-events-none absolute rounded border-2 border-blue-400"
      />
      {/* Step bubble */}
      <div
        style={{ top: measured.pos.top, left: measured.pos.left, width: BUBBLE.width }}
        className="absolute rounded-lg border border-blue-500 bg-zinc-900 p-3 font-mono text-zinc-100 shadow-xl"
      >
        <p className="text-sm font-medium">{t(current.titleKey)}</p>
        <p className="mt-1 text-xs leading-relaxed text-zinc-400">{t(current.bodyKey)}</p>
        <div className="mt-3 flex items-center justify-between gap-2">
          <span className="text-xs text-zinc-500">
            {t('hints.step', { n: state.index + 1, total: steps.length })}
          </span>
          <div className="flex gap-2">
            {!state.isFirst && (
              <button
                type="button"
                onClick={handleBack}
                className="rounded px-2 py-1 text-xs text-zinc-300 transition-colors hover:bg-zinc-800"
              >
                {t('hints.back')}
              </button>
            )}
            <button
              ref={nextRef}
              type="button"
              onClick={handleNext}
              className="rounded bg-blue-600 px-3 py-1 text-xs text-white transition-colors hover:bg-blue-700"
            >
              {state.isLast ? t('hints.gotIt') : t('hints.next')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default HintsWalkthrough
