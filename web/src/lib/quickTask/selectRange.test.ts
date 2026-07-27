import { describe, it, expect } from 'vitest'
import { selectRange } from './selectRange'

const ids = ['a', 'b', 'c', 'd', 'e']

describe('selectRange', () => {
  it('selects forwards inclusively', () => {
    expect(selectRange(ids, 'b', 'd')).toEqual(['b', 'c', 'd'])
  })

  it('selects backwards inclusively', () => {
    expect(selectRange(ids, 'd', 'b')).toEqual(['b', 'c', 'd'])
  })

  it('selects a single item when anchor and target match', () => {
    expect(selectRange(ids, 'c', 'c')).toEqual(['c'])
  })

  it('returns just the target when the anchor is unknown', () => {
    expect(selectRange(ids, 'zzz', 'c')).toEqual(['c'])
  })

  it('returns an empty selection when the target is unknown', () => {
    expect(selectRange(ids, 'a', 'zzz')).toEqual([])
  })

  it('spans the whole list from first to last', () => {
    expect(selectRange(ids, 'a', 'e')).toEqual(ids)
  })
})
