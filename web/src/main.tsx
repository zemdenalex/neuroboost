import React from 'react'
import ReactDOM from 'react-dom/client'
import './i18n'
import App from './App'
import './index.css'
import { applyCurrentTheme, applyTheme, resolveTheme, schemeFromHash, telegramChrome, type Theme } from './lib/theme/theme'
import { loadWebApp } from './lib/telegram/webApp'
import { applyTelegramPalette, paletteFromHash, telegramChromeFrom } from './lib/theme/telegramPalette'

// Before the first frame, so neither a light Telegram nor a light choice
// flashes the dark app. Telegram passes its theme in the launch hash; the
// web's own choice is mirrored on the device (docs/tasks-light-theme.md).
applyCurrentTheme()
// Inside Telegram, the person's own colours on top (MA3b, Denis 26.09: «everything
// from Telegram»); outside it there is no palette and nothing changes.
applyTelegramPalette(paletteFromHash(window.location.hash))
// «System» follows the device when it switches between light and dark.
window.matchMedia?.('(prefers-color-scheme: dark)').addEventListener?.('change', () => applyCurrentTheme())
// And follow it when the person switches Telegram's theme with the app open.
void loadWebApp().then((wa) => {
  if (!wa) return
  const paint = (theme: Theme) => {
    applyTheme(theme)
    const params = wa.themeParams ?? paletteFromHash(window.location.hash)
    applyTelegramPalette(params)
    // Telegram's frame in the page's colours (LT6); older clients lack these.
    const c = telegramChromeFrom(params) ?? telegramChrome(theme)
    wa.setHeaderColor?.(c.header)
    wa.setBackgroundColor?.(c.background)
  }
  paint(resolveTheme({ telegram: wa.colorScheme ?? schemeFromHash(window.location.hash) }))
  wa.onEvent?.('themeChanged', () => {
    if (wa.colorScheme) paint(wa.colorScheme)
  })
})
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
