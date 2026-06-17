import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Sparkles } from 'lucide-react'

/**
 * First-run welcome card. Explains the core loop in one screen and offers two
 * exits: start the guided checklist, or skip. Purely presentational — the
 * caller supplies the handlers (from OnboardingContext).
 */
export function WelcomeCard({
  onStart,
  onSkip,
}: {
  onStart: () => void
  onSkip: () => void
}) {
  const { t } = useTranslation('onboarding')
  const startRef = useRef<HTMLButtonElement>(null)

  // Honour aria-modal: move focus into the dialog on mount and let Escape skip.
  useEffect(() => {
    startRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onSkip()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onSkip])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-label={t('welcome.title')}
        className="w-full max-w-md rounded-xl border border-zinc-700 bg-zinc-900 p-6 font-mono text-zinc-100 shadow-xl"
      >
        <div className="mb-4 flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-600/20 text-blue-400">
            <Sparkles className="h-5 w-5" />
          </span>
          <h2 className="text-lg font-medium">{t('welcome.title')}</h2>
        </div>

        <p className="mb-3 text-sm text-zinc-300">{t('welcome.intro')}</p>
        <p className="mb-6 text-sm text-zinc-400">{t('welcome.loop')}</p>

        <div className="flex flex-col gap-2 sm:flex-row-reverse">
          <button
            ref={startRef}
            onClick={onStart}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
          >
            {t('welcome.getStarted')}
          </button>
          <button
            onClick={onSkip}
            className="rounded-lg px-4 py-2 text-sm text-zinc-400 transition-colors hover:text-zinc-200"
          >
            {t('welcome.skip')}
          </button>
        </div>
      </div>
    </div>
  )
}

export default WelcomeCard
