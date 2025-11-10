import React, { createContext, useContext, useState } from 'react'
type Theme = 'light' | 'dark'
const ThemeContext = createContext<{ theme: Theme; toggle: () => void } | undefined>(undefined)
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>('dark')
  const toggle = () => setTheme(t => (t === 'dark' ? 'light' : 'dark'))
  return <ThemeContext.Provider value={{ theme, toggle }}>{children}</ThemeContext.Provider>
}
export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
