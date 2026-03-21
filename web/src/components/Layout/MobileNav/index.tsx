import { useAuthContext } from '../../../contexts/AuthContext'
import { BottomTabBar } from './BottomTabBar'
import { HamburgerDrawer } from './HamburgerDrawer'
import { FabBottomSheet } from './FabBottomSheet'

type MobileNavType = 'bottom_tabs' | 'hamburger' | 'fab'

export function MobileNavigation() {
  const { user } = useAuthContext()
  const navType = (user?.settings?.mobile_nav as MobileNavType) || 'bottom_tabs'

  switch (navType) {
    case 'hamburger':
      return <HamburgerDrawer />
    case 'fab':
      return <FabBottomSheet />
    default:
      return <BottomTabBar />
  }
}
