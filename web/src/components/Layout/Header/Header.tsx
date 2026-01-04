import { useState, useEffect } from 'react'
import HorizontalHeader from './HorizontalHeader'
import VerticalSidebar from './VerticalSidebar'

type HeaderVariant = 'horizontal' | 'vertical'

export default function Header() {
  // Read preference from localStorage (later: from user settings API)
  const [variant, setVariant] = useState<HeaderVariant>('horizontal')

  useEffect(() => {
    const saved = localStorage.getItem('neuroboost-header-variant') as HeaderVariant
    if (saved === 'horizontal' || saved === 'vertical') {
      setVariant(saved)
    }
  }, [])

  // Listen for changes from Settings page
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'neuroboost-header-variant' && e.newValue) {
        setVariant(e.newValue as HeaderVariant)
      }
    }
    window.addEventListener('storage', handleStorageChange)
    return () => window.removeEventListener('storage', handleStorageChange)
  }, [])

  return variant === 'horizontal' ? <HorizontalHeader /> : <VerticalSidebar />
}

export { HorizontalHeader, VerticalSidebar }