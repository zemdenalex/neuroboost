/** @type {import('tailwindcss').Config} */

// zinc, white and black are CSS variables (src/index.css), so one attribute on
// <html> turns the whole app light for the Telegram Mini App (Denis 25.09:
// "follow Telegram's theme"). In the dark theme the variables hold Tailwind's
// own hex codes, so nothing changes there. Two colours stay fixed on purpose:
// `onaccent` is text on a coloured button, `scrim` the dimming behind a modal.
const v = (name) => `rgb(var(--nb-${name}) / <alpha-value>)`
const zinc = Object.fromEntries(
  [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950].map((s) => [s, v(`zinc-${s}`)])
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
      },
    },
  },
  plugins: [],
}
