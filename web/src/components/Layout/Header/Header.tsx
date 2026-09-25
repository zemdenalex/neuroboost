import { useState, useEffect } from 'react'
import HorizontalHeader from './HorizontalHeader'
import VerticalSidebar from './VerticalSidebar'
import { useMediaQuery } from '../../../hooks/useMediaQuery'
import { effectiveHeaderVariant, PHONE_QUERY } from '../../../lib/layout/headerVariant'

type HeaderVariant = 'horizontal' | 'vertical'

export default function Header() {
  const [variant, setVariant] = useState<HeaderVariant>(() => {
    const saved = localStorage.getItem('neuroboost-header-variant') as HeaderVariant
    return saved === 'vertical' ? 'vertical' : 'horizontal'
  })

  useEffect(() => {
    // Listen for changes from Settings page (same tab)
    const handleLayoutChange = (e: CustomEvent<HeaderVariant>) => {
      setVariant(e.detail)
    }

    // Listen for changes from other tabs
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'neuroboost-header-variant' && e.newValue) {
        setVariant(e.newValue as HeaderVariant)
      }
    }

    window.addEventListener('neuroboost-layout-change', handleLayoutChange as EventListener)
    window.addEventListener('storage', handleStorageChange)
    
    return () => {
      window.removeEventListener('neuroboost-layout-change', handleLayoutChange as EventListener)
      window.removeEventListener('storage', handleStorageChange)
    }
  }, [])

  const isPhone = useMediaQuery(PHONE_QUERY)
  return effectiveHeaderVariant(variant, isPhone) === 'horizontal' ? <HorizontalHeader /> : <VerticalSidebar />
}

export { HorizontalHeader, VerticalSidebar }