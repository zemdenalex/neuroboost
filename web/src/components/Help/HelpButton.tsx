import { useEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { HelpCircle, X } from 'lucide-react'
import { helpKeysFor } from '../../lib/onboarding/helpContent'

/**
 * Always-available contextual help. A "?" trigger (mounted in the header) opens
 * a right-side slide-in panel explaining the current page, resolved from the
 * route via helpKeysFor. Self-contained: holds its own open state.
 */
export function HelpButton() {
  const [open, setOpen] = useState(false)
  const { pathname } = useLocation()
  const { t } = useTranslation('onboarding')
  const { titleKey, bodyKey } = helpKeysFor(pathname)
  const closeRef = useRef<HTMLButtonElement>(null)

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
      <button
        onClick={() => setOpen(true)}
        aria-label={t('help.open')}
        title={t('help.open')}
        className="flex h-8 w-8 items-center justify-center rounded-md text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-white"
      >
        <HelpCircle className="h-5 w-5" />
      </button>

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
          </aside>
        </div>
      )}
    </>
  )
}

export default HelpButton
