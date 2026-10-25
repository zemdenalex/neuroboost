import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Maximize2 } from 'lucide-react'
import { fullscreenSupported, readFullscreenPref, setFullscreen, storeFullscreenPref } from '../../../lib/telegram/fullscreen'
import { launchedInTelegram, loadWebApp, type TgWebApp } from '../../../lib/telegram/webApp'

/**
 * Only inside the Telegram Mini App. Full screen is a per-device choice
 * (localStorage), not an account setting: see lib/telegram/fullscreen.ts.
 */
export function TelegramSection() {
  const { t } = useTranslation('settings')
  const [wa, setWa] = useState<TgWebApp | null>(null)
  const [on, setOn] = useState(() => readFullscreenPref(window.localStorage))

  useEffect(() => {
    if (!launchedInTelegram()) return
    let alive = true
    void loadWebApp().then((app) => alive && setWa(app))
    return () => {
      alive = false
    }
  }, [])

  // The box shows what Telegram really did: leaving full screen from Telegram's
  // own button, or a refusal, unticks it (the saved choice follows on refusal).
  useEffect(() => {
    if (!wa) return
    const follow = () => setOn(Boolean(wa.isFullscreen))
    // Leaving through Telegram's own button is the person's choice too.
    const left = () => {
      follow()
      storeFullscreenPref(window.localStorage, Boolean(wa.isFullscreen))
    }
    wa.onEvent?.('fullscreenChanged', left)
    wa.onEvent?.('fullscreenFailed', follow)
    return () => {
      wa.offEvent?.('fullscreenChanged', left)
      wa.offEvent?.('fullscreenFailed', follow)
    }
  }, [wa])

  if (!launchedInTelegram() || !wa) return null
  const supported = fullscreenSupported(wa)

  const change = (next: boolean) => {
    setOn(next)
    storeFullscreenPref(window.localStorage, next)
    setFullscreen(wa, next)
  }

  return (
    <section data-testid="settings-telegram" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Maximize2 className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('telegram.title')}</h2>
      </div>
      <label className="flex items-center gap-3 text-sm font-mono text-zinc-200">
        <input
          type="checkbox"
          data-testid="telegram-fullscreen"
          checked={on && supported}
          disabled={!supported}
          onChange={(e) => change(e.target.checked)}
        />
        {t('telegram.fullscreen')}
      </label>
      <p className="text-xs text-zinc-500 mt-3">{supported ? t('telegram.fullscreen.hint') : t('telegram.fullscreen.unsupported')}</p>
    </section>
  )
}
