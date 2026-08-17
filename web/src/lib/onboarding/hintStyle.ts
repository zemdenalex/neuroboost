export type HintStyle = 'bubbles' | 'walkthrough' | 'markers'

const HINT_STYLE_KEY = 'neuroboost-hints-style'
const VALID: readonly HintStyle[] = ['bubbles', 'walkthrough', 'markers']
const DEFAULT_HINT_STYLE: HintStyle = 'bubbles'

export function parseHintStyle(raw: string | null): HintStyle {
  return VALID.includes(raw as HintStyle) ? (raw as HintStyle) : DEFAULT_HINT_STYLE
}

export function getHintStyle(storage: Storage = localStorage): HintStyle {
  try {
    return parseHintStyle(storage.getItem(HINT_STYLE_KEY))
  } catch {
    return DEFAULT_HINT_STYLE
  }
}

export function setHintStyle(style: HintStyle, storage: Storage = localStorage): void {
  try {
    storage.setItem(HINT_STYLE_KEY, style)
  } catch {
    // preference is non-critical; ignore quota/availability errors
  }
}
