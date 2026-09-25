import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { backButtonVisible, loadWebApp, takeStartRoute } from './webApp'

/**
 * Inside the Mini App, Telegram's own BackButton (top-left of its frame) steps
 * back on inner pages; on the bottom-bar tabs it is hidden and Telegram's close
 * button is the way out. Outside Telegram this does nothing: loadWebApp()
 * resolves null there without loading anything.
 */
export function useTelegramBackButton(): void {
  const { pathname } = useLocation()
  const navigate = useNavigate()
  // A start link (t.me/<bot>/<app>?startapp=t-<id>) opens its page once.
  useEffect(() => {
    const to = takeStartRoute()
    if (to) navigate(to, { replace: true })
  }, [navigate])
  useEffect(() => {
    let off: (() => void) | undefined
    let live = true
    void loadWebApp().then((wa) => {
      if (!wa || !live) return
      if (!backButtonVisible(pathname)) {
        wa.BackButton.hide()
        return
      }
      const back = () => navigate(-1)
      wa.BackButton.onClick(back)
      wa.BackButton.show()
      off = () => wa.BackButton.offClick(back)
    })
    return () => {
      live = false
      off?.()
    }
  }, [pathname, navigate])
}
