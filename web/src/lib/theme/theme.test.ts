import { describe, it, expect } from 'vitest'
import { resolveTheme, schemeFromHash, schemeFromBg } from './theme'

describe('schemeFromBg', () => {
  it('reads a light or dark background colour', () => {
    expect(schemeFromBg('#ffffff')).toBe('light')
    expect(schemeFromBg('#f1f1f1')).toBe('light')
    expect(schemeFromBg('#212d3b')).toBe('dark')
    expect(schemeFromBg('#000')).toBe('dark')
  })
  it('is null for anything that is not a colour', () => {
    expect(schemeFromBg(undefined)).toBeNull()
    expect(schemeFromBg('blue')).toBeNull()
    expect(schemeFromBg('#12')).toBeNull()
  })
})

describe('schemeFromHash (Telegram launch)', () => {
  const params = (bg: string) =>
    '#tgWebAppData=x&tgWebAppThemeParams=' + encodeURIComponent(JSON.stringify({ bg_color: bg, text_color: '#000000' }))
  it('takes the scheme from the theme Telegram passed at launch', () => {
    expect(schemeFromHash(params('#ffffff'))).toBe('light')
    expect(schemeFromHash(params('#17212b'))).toBe('dark')
  })
  it('is null outside Telegram or without usable params', () => {
    expect(schemeFromHash('')).toBeNull()
    expect(schemeFromHash('#tgWebAppData=x')).toBeNull()
    expect(schemeFromHash('#tgWebAppData=x&tgWebAppThemeParams=%7Bbroken')).toBeNull()
  })
})

describe('resolveTheme (Denis 25.09: follow Telegram)', () => {
  it('inside Telegram follows its scheme', () => {
    expect(resolveTheme({ telegram: 'light' })).toBe('light')
    expect(resolveTheme({ telegram: 'dark' })).toBe('dark')
  })
  it('outside Telegram stays dark, as the web has always been', () => {
    expect(resolveTheme({ telegram: null })).toBe('dark')
  })
})
