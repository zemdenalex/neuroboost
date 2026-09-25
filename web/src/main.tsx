import React from 'react'
import ReactDOM from 'react-dom/client'
import './i18n'
import App from './App'
import './index.css'
import { applyCurrentTheme, applyTheme, resolveTheme, schemeFromHash, telegramChrome, type Theme } from './lib/theme/theme'
import { loadWebApp } from './lib/telegram/webApp'

// Before the first frame, so neither a light Telegram nor a light choice
// flashes the dark app. Telegram passes its theme in the launch hash; the
// web's own choice is mirrored on the device (docs/tasks-light-theme.md).
applyCurrentTheme()
// «System» follows the device when it switches between light and dark.
window.matchMedia?.('(prefers-color-scheme: dark)').addEventListener?.('change', () => applyCurrentTheme())
// And follow it when the person switches Telegram's theme with the app open.
void loadWebApp().then((wa) => {
  if (!wa) return
  const paint = (theme: Theme) => {
    applyTheme(theme)
    // Telegram's frame in the app's colours (LT6); older clients lack these.
    const c = telegramChrome(theme)
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
