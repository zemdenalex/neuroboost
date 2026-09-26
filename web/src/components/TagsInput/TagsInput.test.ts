import { describe, it, expect } from 'vitest'
import { parseTags } from './TagsInput'

describe('parseTags', () => {
  it('trims, drops empty parts and repeats', () => {
    expect(parseTags(' дом, работа ,, дом,')).toEqual(['дом', 'работа'])
  })
})
