import type { TimerMode } from '../../lib/pomodoro/types'

export function formatMs(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

export const MODE_HEX: Record<TimerMode, string> = {
  work: '#ef4444',
  shortBreak: '#22c55e',
  longBreak: '#3b82f6',
}

export const MODE_BTN: Record<TimerMode, string> = {
  work: 'bg-red-500',
  shortBreak: 'bg-green-500',
  longBreak: 'bg-blue-500',
}
