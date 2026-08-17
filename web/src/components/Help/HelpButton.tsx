import { useEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { HelpCircle, Lightbulb, X } from 'lucide-react'
import { helpKeysFor } from '../../lib/onboarding/helpContent'
import { hintsForRoute } from '../../lib/onboarding/hintsContent'
import { useOnboarding } from '../../contexts/OnboardingContext'

/**
 * Always-available contextual help. A trigger opens a right-side slide-in panel
 * explaining the current page, resolved from the route via helpKeysFor.
 * Self-contained: holds its own open state.
 *
 * The panel is shared; only the trigger's appearance changes by variant:
 *  - "icon" (default): compact "?" button for the horizontal header.
 *  - "sidebar": full-width labelled row matching the vertical sidebar nav idiom.
 */
export function HelpButton({ variant = 'icon' }: { variant?: 'icon' | 'sidebar' }) {
  const [open, setOpen] = useState(false)
  const { pathname } = useLocation()
  const { t } = useTranslation('onboarding')
  const { titleKey, bodyKey } = helpKeysFor(pathname)
  const { hintStyle, showHints } = useOnboarding()
  const closeRef = useRef<HTMLButtonElement>(null)

  // "Show hints" reveals the on-demand bubbles/walkthrough; pointless when the
  // style is always-on markers, or when the page has no anchored hints.
  const canShowHints = hintStyle !== 'markers' && hintsForRoute(pathname).length > 0

  const handleShowHints = () => {
    setOpen(false)
    showHints()
  }

  useEffect(() => {
    if (!open) return
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])

  return (
    <>
      {variant === 'sidebar' ? (
        <button
          onClick={() => setOpen(true)}
          aria-label={t('help.open')}
          className="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-mono text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
        >
          <HelpCircle className="w-4 h-4" />
          {t('help.label')}
        </button>
      ) : (
        <button
          onClick={() => setOpen(true)}
          aria-label={t('help.open')}
          title={t('help.open')}
          className="flex h-8 w-8 items-center justify-center rounded-md text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-white"
        >
          <HelpCircle className="h-5 w-5" />
        </button>
      )}

      {open && (
        <div
          className="fixed inset-0 z-50 flex justify-end bg-black/50"
          onClick={() => setOpen(false)}
        >
          <aside
            role="dialog"
            aria-modal="true"
            aria-label={t(titleKey)}
            onClick={(e) => e.stopPropagation()}
            className="h-full w-full max-w-sm overflow-y-auto border-l border-zinc-700 bg-zinc-900 p-6 font-mono text-zinc-100 shadow-xl"
          >
            <div className="mb-4 flex items-center justify-between gap-3">
              <h2 className="text-base font-medium">{t(titleKey)}</h2>
              <button
                ref={closeRef}
                onClick={() => setOpen(false)}
                aria-label={t('help.close')}
                className="flex-none text-zinc-500 transition-colors hover:text-zinc-200"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <p className="whitespace-pre-line text-sm leading-relaxed text-zinc-300">{t(bodyKey)}</p>
            {canShowHints && (
              <button
                type="button"
                onClick={handleShowHints}
                className="mt-5 flex w-full items-center justify-center gap-2 rounded-lg border border-blue-500 bg-blue-600/20 px-4 py-2 text-sm text-blue-300 transition-colors hover:bg-blue-600/30"
              >
                <Lightbulb className="h-4 w-4" />
                {t('hints.showHints')}
              </button>
            )}
          </aside>
        </div>
      )}
    </>
  )
}

export default HelpButton
