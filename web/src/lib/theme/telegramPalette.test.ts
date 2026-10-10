import { describe, expect, it } from 'vitest'
import { contrastRatio, paletteFromHash, telegramChromeFrom, telegramPaletteVars } from './telegramPalette'

// Telegram's Android night theme, as sent in themeParams.
const NIGHT = {
  bg_color: '#212d3b',
  secondary_bg_color: '#1d2733',
  text_color: '#ffffff',
  hint_color: '#7d8b99',
  link_color: '#5eabe1',
  button_color: '#50a8eb',
  button_text_color: '#ffffff',
}

describe('telegramPaletteVars (MA3b, Denis 26.09: variant 2, everything from Telegram)', () => {
  it('puts the page on bg_color, panels on secondary_bg_color and text on text_color', () => {
    const vars = telegramPaletteVars(NIGHT)!
    expect(vars['--nb-zinc-950']).toBe('33 45 59')
    expect(vars['--nb-zinc-900']).toBe('29 39 51')
    expect(vars['--nb-white']).toBe('255 255 255')
    expect(vars['--nb-black']).toBe('33 45 59')
    expect(vars['--nb-zinc-500']).toBe('125 139 153')
  })

  it('takes buttons from button_color and link text from link_color', () => {
    const vars = telegramPaletteVars(NIGHT)!
    expect(vars['--nb-blue-600']).toBe('80 168 235')
    expect(vars['--nb-blue-500']).toBe('80 168 235')
    expect(vars['--nb-blue-400']).toBe('94 171 225')
  })

  it('fills the scale between page and text in order, so borders stay fainter than text', () => {
    const vars = telegramPaletteVars(NIGHT)!
    const lum = (k: string) => vars[k].split(' ').map(Number).reduce((a, b) => a + b, 0)
    const steps = ['--nb-zinc-950', '--nb-zinc-800', '--nb-zinc-700', '--nb-zinc-600', '--nb-zinc-400', '--nb-zinc-300', '--nb-zinc-200', '--nb-zinc-100', '--nb-zinc-50']
    for (let i = 1; i < steps.length; i++) expect(lum(steps[i])).toBeGreaterThan(lum(steps[i - 1]))
  })

  it('falls back to our palette when text on the page would be hard to read', () => {
    expect(telegramPaletteVars({ ...NIGHT, text_color: '#3a4656' })).toBeNull()
  })

  it('keeps our accent when white text on the Telegram button would be hard to read', () => {
    const vars = telegramPaletteVars({ ...NIGHT, button_color: '#e8f0ff' })!
    expect(vars['--nb-zinc-950']).toBe('33 45 59')
    expect(vars['--nb-blue-600']).toBeUndefined()
  })

  it('does nothing without a page and text colour, or with a colour that is not hex', () => {
    expect(telegramPaletteVars({})).toBeNull()
    expect(telegramPaletteVars({ bg_color: 'red', text_color: '#fff' })).toBeNull()
    expect(telegramPaletteVars(undefined)).toBeNull()
  })

  it('derives panels from the page when secondary_bg_color is missing', () => {
    const { secondary_bg_color: _drop, ...rest } = NIGHT
    const vars = telegramPaletteVars(rest)!
    expect(vars['--nb-zinc-900']).not.toBe(vars['--nb-zinc-950'])
  })
})

describe('telegramChromeFrom', () => {
  it('paints Telegram\'s header and background in the same colours as the page', () => {
    expect(telegramChromeFrom(NIGHT)).toEqual({ header: '#1d2733', background: '#212d3b' })
    expect(telegramChromeFrom({})).toBeNull()
  })
})

describe('paletteFromHash', () => {
  it('reads tgWebAppThemeParams from the launch hash, before Telegram\'s script', () => {
    const hash = '#tgWebAppData=x&tgWebAppThemeParams=' + encodeURIComponent(JSON.stringify(NIGHT))
    expect(paletteFromHash(hash)).toEqual(NIGHT)
    expect(paletteFromHash('#tgWebAppThemeParams=not-json')).toBeNull()
    expect(paletteFromHash('')).toBeNull()
  })
})

describe('contrastRatio', () => {
  it('is 21 for black on white and 1 for a colour on itself', () => {
    expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 0)
    expect(contrastRatio('#50a8eb', '#50a8eb')).toBeCloseTo(1, 5)
  })
})
