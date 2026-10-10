import { describe, expect, it } from 'vitest'
import { matchesStatusFilter } from './statusFilter'

// A scheduled task is still to be done: it stays under «К выполнению», marked
// with its time — as the bot shows it (pass 3, A15, Denis 23.09: «задача
// осталась в списке с 📅»). Found by the row-4 agent on 26.09.
describe('matchesStatusFilter', () => {
  it('keeps a scheduled task under «to do»', () => {
    expect(matchesStatusFilter('SCHEDULED', 'TODO')).toBe(true)
  })

  it('still filters everything else by its own status', () => {
    expect(matchesStatusFilter('TODO', 'TODO')).toBe(true)
    expect(matchesStatusFilter('DONE', 'TODO')).toBe(false)
    expect(matchesStatusFilter('TODO', 'SCHEDULED')).toBe(false)
    expect(matchesStatusFilter('DONE', 'ALL')).toBe(true)
  })
})
