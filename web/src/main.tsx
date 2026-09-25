import React from 'react'
import ReactDOM from 'react-dom/client'
import './i18n'
import App from './App'
import './index.css'
import { applyTheme, resolveTheme, schemeFromHash } from './lib/theme/theme'
import { loadWebApp } from './lib/telegram/webApp'

// Before the first frame, so a light Telegram never flashes the dark app.
// Telegram passes its theme in the launch hash (docs/tasks-light-theme.md).
applyTheme(resolveTheme({ telegram: schemeFromHash(window.location.hash) }))
// And follow it when the person switches Telegram's theme with the app open.
void loadWebApp().then((wa) => {
  if (!wa?.onEvent) return
  wa.onEvent('themeChanged', () => {
    if (wa.colorScheme) applyTheme(wa.colorScheme)
  })
})
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
