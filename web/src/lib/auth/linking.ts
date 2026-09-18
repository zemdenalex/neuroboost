/**
 * Which half of the account-linking block a person should see.
 *
 * Pure, and separate from the component, because the component cannot be
 * tested here — this repository has no component-testing setup, and adding one
 * for three branches would be a larger change than the feature. The decision is
 * the part that can be wrong; the markup around it is not.
 */
export type LinkingState = 'linked' | 'offer-telegram' | 'offer-credentials'

export function linkingState(user: { email?: string; tg_id?: number } | null | undefined): LinkingState {
  const hasEmail = Boolean(user?.email)
  const hasTelegram = Boolean(user?.tg_id)
  if (hasEmail && hasTelegram) return 'linked'
  // 🔴 An account with NEITHER cannot exist — every account arrives through one
  // of the two doors — but if one ever did, offering credentials is the safe
  // answer: it creates a way in, where offering the Telegram code would ask for
  // a secret from a bot this person has never opened.
  return hasEmail ? 'offer-telegram' : 'offer-credentials'
}

/**
 * The six digits, as they must reach the API.
 *
 * 🔴 Kept as a STRING throughout. «048215» read as a number is 48215, which
 * matches nothing, and the defect appears for one person in ten — the nine
 * whose code has no leading zero see a feature that works.
 */
export function normalizeLinkCode(raw: string): string {
  return raw.replace(/\D/g, '').slice(0, 6)
}
