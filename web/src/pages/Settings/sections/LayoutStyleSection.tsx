import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { LayoutGrid } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { showToast } from '../../../components/ui/Toast'

type HeaderVariant = 'horizontal' | 'vertical'

const VARIANTS: HeaderVariant[] = ['horizontal', 'vertical']

/**
 * Horizontal or vertical navigation.
 *
 * Step 4 of the Settings split (2026-08-14). Reads its own slice of the user
 * rather than being handed it: the parent's single `useEffect([user])` used to
 * resync eight fields and skip three others, and that asymmetry is exactly what
 * a 866-line file with one shared effect produces. A section that owns its own
 * value cannot drift from the others, because there are no others.
 *
 * Saves immediately rather than through the debounce — this is a two-value
 * choice, not a slider, so there is nothing to coalesce.
 *
 * ⚠ A failure shows the same neutral toast as a success. That is the project's
 * existing convention (Calendars/CalendarsSection.tsx reports its errors the
 * same way) and is preserved here rather than quietly diverging — but it is a
 * real gap: Toast has no error variant, so "saved" and "could not save" look
 * identical.
 */
export function LayoutStyleSection() {
  const { t } = useTranslation('settings')
  const { user, updateSettings } = useAuthContext()
  const [style, setStyle] = useState<HeaderVariant>(user?.settings?.header_variant ?? 'horizontal')

  // Resync when the profile arrives or changes — a cold open renders before
  // the user is loaded, and refreshUser() can bring a change made elsewhere.
  useEffect(() => {
    if (user?.settings?.header_variant) setStyle(user.settings.header_variant)
  }, [user])

  const change = async (next: HeaderVariant) => {
    // Optimistic: the layout switches under the user's hand, and reverting on
    // failure would be more jarring than the toast that reports it.
    setStyle(next)
    try {
      await updateSettings({ header_variant: next })
      showToast(t('saved'))
    } catch {
      showToast(t('error.saveLayout'))
    }
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <LayoutGrid className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('layout.title')}</h2>
      </div>

      <div className="flex gap-3">
        {VARIANTS.map((v) => (
          <button
            key={v}
            onClick={() => change(v)}
            className={`flex-1 p-3 rounded-lg border text-sm font-mono transition-colors ${
              style === v
                ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
            }`}
          >
            {t(`layout.${v}`)}
          </button>
        ))}
      </div>
      <p className="text-xs text-zinc-500 mt-2">{t('layout.note')}</p>
    </section>
  )
}
