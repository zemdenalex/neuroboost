import { useEffect } from 'react'
import { useAuthContext } from '../contexts/AuthContext'

/**
 * Reads the user's ui_scale setting and applies it as the root font-size
 * so every rem-based measurement scales proportionally. Called once in
 * App.tsx so the scale is active on every page, not only on Settings.
 */
export function useUIScale() {
  const { user } = useAuthContext()
  const scale = user?.settings?.ui_scale ?? 100

  useEffect(() => {
    document.documentElement.style.fontSize = `${scale}%`
  }, [scale])
}
