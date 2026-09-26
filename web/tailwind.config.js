/** @type {import('tailwindcss').Config} */
import plugin from 'tailwindcss/plugin'
import palette from 'tailwindcss/colors'

// zinc, white and black are CSS variables (src/index.css), so one attribute on
// <html> turns the whole app light for the Telegram Mini App (Denis 25.09:
// "follow Telegram's theme"). In the dark theme the variables hold Tailwind's
// own hex codes, so nothing changes there. Two colours stay fixed on purpose:
// `onaccent` is text on a coloured button, `scrim` the dimming behind a modal.
const v = (name) => `rgb(var(--nb-${name}) / <alpha-value>)`
const zinc = Object.fromEntries(
  [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950].map((s) => [s, v(`zinc-${s}`)])
)

// Accent colours in the shades tuned for a dark page flip in the light theme,
// like zinc: pale text (300, 400) darkens, dark tints (800, 900, 950) lighten.
// The middle shades (500-700: buttons, dots) are the same in both themes.
const ACCENTS = ['red', 'green', 'blue', 'amber', 'yellow', 'purple', 'emerald', 'orange', 'indigo', 'rose', 'sky', 'teal', 'violet', 'pink', 'cyan', 'lime']
const FLIP = { 300: 700, 400: 600, 800: 200, 900: 100, 950: 50 }
const rgb = (hex) => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(' ')
// Blue's middle shades (buttons, focus rings) are variables too, the same hex
// in both themes, so the Mini App can put the person's Telegram button colour
// there (MA3b, src/lib/theme/telegramPalette.ts).
const FIXED = { blue: [500, 600, 700] }
const accentColors = Object.fromEntries(
  ACCENTS.map((c) => [
    c,
    Object.fromEntries([...Object.keys(FLIP), ...(FIXED[c] ?? [])].map((s) => [s, v(`${c}-${s}`)])),
  ])
)
const fixedVars = Object.fromEntries(
  Object.entries(FIXED).flatMap(([c, shades]) => shades.map((s) => [`--nb-${c}-${s}`, rgb(palette[c][s])]))
)
const accentVars = (light) =>
  Object.fromEntries(
    ACCENTS.flatMap((c) =>
      Object.entries(FLIP).map(([s, flipped]) => [`--nb-${c}-${s}`, rgb(palette[c][light ? flipped : s])])
    )
  )

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        zinc,
        white: v('white'),
        black: v('black'),
        onaccent: '#ffffff',
        scrim: '#000000',
        ...accentColors,
      },
    },
  },
  plugins: [
    plugin(({ addBase }) => {
      addBase({ ':root': { ...fixedVars, ...accentVars(false) }, ':root[data-theme="light"]': accentVars(true) })
    }),
  ],
}
