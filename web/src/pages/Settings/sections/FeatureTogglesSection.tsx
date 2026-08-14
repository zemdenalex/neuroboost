import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Sliders } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import type { UserSettings } from '../../../api/auth'

/**
 * The default set. Also the list of what the page offers, since the toggles are
 * rendered from the object's own keys.
 *
 * 🔴 `opportunities`, `needs` and `graph` no longer have anything behind them.
 * The backend packages answering those routes were 501 stubs with no writer and
 * were deleted on 2026-08-14, along with the dead GraphView tree on the front
 * end. The switches still flip and still persist — they simply enable nothing.
 *
 * Left in place deliberately rather than quietly dropped: removing them changes
 * what the user sees and what is stored in their settings, which is a product
 * decision, not a side effect of a refactor. Recorded here so the next reader
 * does not have to rediscover it.
 */
const DEFAULT_FEATURES = {
  dreams: false,
  goals: false,
  projects: false,
  opportunities: false,
  needs: false,
  graph: false,
  timeline: false,
  tools: true,
}

type Features = typeof DEFAULT_FEATURES

interface Props {
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * Which optional views the app offers.
 *
 * Step 10 of the Settings split (2026-08-14).
 */
export function FeatureTogglesSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [features, setFeatures] = useState<Features>(() => ({
    ...DEFAULT_FEATURES,
    ...(user?.settings?.features ?? {}),
  }))

  // Merged over the defaults, not replaced: a stored object written before a
  // toggle existed would otherwise leave that key undefined and render a switch
  // in neither state.
  useEffect(() => {
    if (user?.settings?.features) {
      setFeatures((prev) => ({ ...prev, ...user.settings!.features }))
    }
  }, [user])

  const toggle = (key: keyof Features) => {
    const next = { ...features, [key]: !features[key] }
    setFeatures(next)
    autoSave({ features: next })
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Sliders className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('featureToggles.title')}</h2>
      </div>

      <div className="grid grid-cols-2 gap-3">
        {(Object.keys(features) as Array<keyof Features>).map((key) => {
          const enabled = features[key]
          return (
            <button
              key={key}
              onClick={() => toggle(key)}
              className="flex items-center justify-between p-3 bg-zinc-800/50 rounded-lg hover:bg-zinc-800 transition-colors"
            >
              <span className="text-sm text-zinc-300">{t(`featureToggles.${key}View`)}</span>
              <div
                className={`relative w-10 h-5 rounded-full transition-colors ${
                  enabled ? 'bg-blue-600' : 'bg-zinc-600'
                }`}
              >
                <div
                  className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                    enabled ? 'translate-x-5' : 'translate-x-0.5'
                  }`}
                />
              </div>
            </button>
          )
        })}
      </div>
    </section>
  )
}
