import { useLocation } from 'react-router-dom'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { PillWidget } from './PillWidget'
import { CardWidget } from './CardWidget'
import { BarWidget } from './BarWidget'

export function PomodoroWidget() {
  const { isActive, settings } = usePomodoro()
  const { pathname } = useLocation()

  if (!isActive) return null
  if (pathname.startsWith('/tools/pomodoro')) return null

  switch (settings.widgetStyle) {
    case 'pill':
      return <PillWidget />
    case 'bar':
      return <BarWidget />
    case 'card':
    default:
      return <CardWidget />
  }
}
