import { describe, it, expect } from 'vitest'
import { readThemeChoice, resolveTheme, schemeFromHash, schemeFromBg, telegramChrome } from './theme'

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

describe('resolveTheme (Denis 25.09: follow Telegram; dark / light / system on the web)', () => {
  it('inside Telegram follows its scheme, whatever was chosen on the web', () => {
    expect(resolveTheme({ telegram: 'light', choice: 'dark' })).toBe('light')
    expect(resolveTheme({ telegram: 'dark', choice: 'light' })).toBe('dark')
  })
  it('outside Telegram takes the choice', () => {
    expect(resolveTheme({ telegram: null, choice: 'light' })).toBe('light')
    expect(resolveTheme({ telegram: null, choice: 'dark' })).toBe('dark')
  })
  it('«system» follows the device', () => {
    expect(resolveTheme({ telegram: null, choice: 'system', systemDark: false })).toBe('light')
    expect(resolveTheme({ telegram: null, choice: 'system', systemDark: true })).toBe('dark')
  })
  it('with no choice stays dark, as the web has always been', () => {
    expect(resolveTheme({ telegram: null })).toBe('dark')
  })
})

describe('readThemeChoice', () => {
  it('reads dark, light or system; anything else is dark', () => {
    expect(readThemeChoice({ theme: 'light' })).toBe('light')
    expect(readThemeChoice({ theme: 'system' })).toBe('system')
    expect(readThemeChoice({ theme: 'sepia' })).toBe('dark')
    expect(readThemeChoice(undefined)).toBe('dark')
  })
})

describe('telegramChrome (LT6)', () => {
  it("paints Telegram's frame in the app's zinc-900 / zinc-950 of the theme", () => {
    expect(telegramChrome('dark')).toEqual({ header: '#18181b', background: '#09090b' })
    expect(telegramChrome('light')).toEqual({ header: '#f4f4f5', background: '#fafafa' })
  })
})
