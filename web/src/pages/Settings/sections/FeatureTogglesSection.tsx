import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Sliders } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { mergeFeatures, DEFAULT_FEATURES, type Features, type FeatureKey } from '../../../lib/settings/features'
import type { UserSettings } from '../../../api/auth'

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
  const [features, setFeatures] = useState<Features>(() => mergeFeatures(user?.settings?.features))

  // Merged over the defaults, not replaced: a stored object written before a
  // toggle existed would otherwise leave that key undefined and render a switch
  // in neither state.
  useEffect(() => {
    if (user?.settings?.features) setFeatures(mergeFeatures(user.settings.features))
  }, [user])

  const toggle = (key: FeatureKey) => {
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
        {(Object.keys(DEFAULT_FEATURES) as FeatureKey[]).map((key) => {
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
