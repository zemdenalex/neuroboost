import { describe, it, expect } from 'vitest'
import { nextParentId, type TrailEntry } from './indent'

describe('nextParentId', () => {
  it('returns undefined when nothing has been created yet', () => {
    expect(nextParentId([], 'in')).toBeUndefined()
    expect(nextParentId([], 'out')).toBeUndefined()
  })

  it('indenting makes the last created task the parent', () => {
    const trail: TrailEntry[] = [{ id: 'a' }]
    expect(nextParentId(trail, 'in')).toBe('a')
  })

  it('indenting twice nests under the newest child', () => {
    const trail: TrailEntry[] = [{ id: 'a' }, { id: 'b', parentId: 'a' }]
    expect(nextParentId(trail, 'in')).toBe('b')
  })

  it('outdenting climbs to the grandparent', () => {
    const trail: TrailEntry[] = [{ id: 'a' }, { id: 'b', parentId: 'a' }, { id: 'c', parentId: 'b' }]
    expect(nextParentId(trail, 'out')).toBe('a')
  })

  it('outdenting from the top level stays at the top level', () => {
    const trail: TrailEntry[] = [{ id: 'a' }]
    expect(nextParentId(trail, 'out')).toBeUndefined()
  })

  it('outdenting from a first-level child returns to the top level', () => {
    const trail: TrailEntry[] = [{ id: 'a' }, { id: 'b', parentId: 'a' }]
    expect(nextParentId(trail, 'out')).toBeUndefined()
  })
})
