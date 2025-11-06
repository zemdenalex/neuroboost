import HorizontalHeader from './HorizontalHeader'
import VerticalSidebar from './VerticalSidebar'
export default function Header() {
  // TODO: read preference to toggle
  const variant:'horizontal'|'vertical' = 'horizontal'
  return variant==='horizontal'? <HorizontalHeader/> : <VerticalSidebar/>
}
