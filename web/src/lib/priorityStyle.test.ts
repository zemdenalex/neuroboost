import { describe, expect, it } from 'vitest'
import { priorityMark, readPriorityStyle } from './priorityStyle'

// Gap list row 16 (Denis 26.09: yes, the web follows the bot's setting).
// bot/internal/format/priority.go: circles (default), dot («●1 ○3», the digit
// keeps the order without colour; 1 = most urgent), dash.
describe('priority style', () => {
  it('reads the bot setting, circles by default', () => {
    expect(readPriorityStyle({ bot: { priority_style: 'dot' } })).toBe('dot')
    expect(readPriorityStyle({ bot: { priority_style: 'dash' } })).toBe('dash')
    expect(readPriorityStyle({ bot: {} })).toBe('circles')
    expect(readPriorityStyle({ bot: { priority_style: 'sparkles' } })).toBe('circles')
    expect(readPriorityStyle(undefined)).toBe('circles')
  })

  it('draws the three styles as the bot does', () => {
    expect(priorityMark('circles', 1)).toEqual({ kind: 'dot', className: 'bg-red-600' })
    expect(priorityMark('dot', 1)).toEqual({ kind: 'text', text: '●1' })
    expect(priorityMark('dot', 2)).toEqual({ kind: 'text', text: '●2' })
    expect(priorityMark('dot', 3)).toEqual({ kind: 'text', text: '○3' })
    expect(priorityMark('dot', 0)).toEqual({ kind: 'text', text: '·0' })
    expect(priorityMark('dash', 4)).toEqual({ kind: 'text', text: '—' })
  })
})
