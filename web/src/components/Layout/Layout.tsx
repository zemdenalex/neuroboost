import React, { useState, useEffect } from 'react'
import Header from './Header'

type HeaderVariant = 'horizontal' | 'vertical'

export function Layout({ children }: { children: React.ReactNode }) {
  const [variant, setVariant] = useState<HeaderVariant>('horizontal')

  useEffect(() => {
    const saved = localStorage.getItem('neuroboost-header-variant') as HeaderVariant
    if (saved === 'horizontal' || saved === 'vertical') {
      setVariant(saved)
    }
  }, [])

  // Listen for changes
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'neuroboost-header-variant' && e.newValue) {
        setVariant(e.newValue as HeaderVariant)
      }
    }
    window.addEventListener('storage', handleStorageChange)
    return () => window.removeEventListener('storage', handleStorageChange)
  }, [])

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 font-mono">
      <Header />
      <main
        className={
          variant === 'horizontal'
            ? 'pt-14' // Top padding for horizontal header
            : 'pl-56' // Left padding for vertical sidebar
        }
      >
        {children}
      </main>
    </div>
  )
}

export default Layout