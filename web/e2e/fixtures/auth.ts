import { test as base, request as playwrightRequest, type Page } from '@playwright/test'
import { createHash, createHmac } from 'node:crypto'

/**
 * A logged-in page, for specs that need to see the app rather than the landing.
 *
 * The session is minted through the same Telegram Login Widget endpoint the
 * bot uses, not by typing a password: no credential has to live in the repo or
 * in CI, and the signature is reproducible from the bot token alone.
 *
 * Requires E2E_TG_BOT_TOKEN and E2E_TG_ID. Without them the fixture skips the
 * test rather than failing it — a machine with no secrets should report "not
 * run", never a red that gets ignored into meaninglessness.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

export const TOKEN_KEY = 'nb_token'
// Unix SECONDS, matching the API's expires_at verbatim. isTokenExpired() in
// web/src/api/client.ts multiplies by 1000 before comparing to Date.now(), so
// writing milliseconds here would push expiry ~55,000 years out and quietly
// mask a genuinely dead session.
export const TOKEN_EXPIRY_KEY = 'nb_token_expiry'

interface TelegramIdentity {
  id: number
  first_name: string
  last_name?: string
  username?: string
}

/**
 * Signs an identity the way Telegram's widget does: sorted `key=value` lines
 * joined by newlines, HMAC-SHA256 under SHA256(botToken).
 *
 * Empty optional fields are omitted, because the server rebuilds its check
 * string only from the fields that arrived.
 */
export function signTelegramPayload(botToken: string, identity: TelegramIdentity, authDate: number) {
  const fields: Record<string, string> = {
    id: String(identity.id),
    first_name: identity.first_name,
    auth_date: String(authDate),
  }
  if (identity.last_name) fields.last_name = identity.last_name
  if (identity.username) fields.username = identity.username

  const dataCheckString = Object.keys(fields)
    .sort()
    .map((k) => `${k}=${fields[k]}`)
    .join('\n')

  const secret = createHash('sha256').update(botToken).digest()
  const hash = createHmac('sha256', secret).update(dataCheckString).digest('hex')

  return { ...fields, id: identity.id, auth_date: authDate, hash }
}

export interface Session {
  token: string
  expiresAt: number
}

export async function mintSession(): Promise<Session> {
  const botToken = process.env.E2E_TG_BOT_TOKEN
  const tgId = process.env.E2E_TG_ID
  if (!botToken || !tgId) {
    throw new Error('E2E_TG_BOT_TOKEN and E2E_TG_ID are required to mint a session')
  }

  const payload = signTelegramPayload(
    botToken,
    { id: Number(tgId), first_name: process.env.E2E_TG_FIRST_NAME ?? 'E2E' },
    Math.floor(Date.now() / 1000)
  )

  const ctx = await playwrightRequest.newContext()
  try {
    const res = await ctx.post(`${API_BASE}/api/auth/telegram`, { data: payload })
    if (!res.ok()) {
      throw new Error(`login failed: ${res.status()} ${await res.text()}`)
    }
    const body = await res.json()
    return { token: body.data.token, expiresAt: body.data.expires_at }
  } finally {
    await ctx.dispose()
  }
}

// Onboarding flags, from web/src/lib/onboarding/onboardingFlag.ts. The welcome
// card is a full-screen overlay on a fresh account and it intercepts every
// click on the calendar underneath — the first run of the recurring-scope spec
// failed on exactly that, and only the failure screenshot said why.
//
// Seeded rather than clicked away: a click is a race against the overlay's own
// mount, and dismissing it is not what these specs are about.
export const WELCOME_SEEN_KEY = 'neuroboost-onboarding-welcome-seen'
export const CHECKLIST_DISMISSED_KEY = 'neuroboost-onboarding-checklist-dismissed'

/**
 * Seeds the session before any page script runs. AuthContext reads localStorage
 * on mount, so writing it after navigation would lose the race and bounce the
 * test to /login — which looks identical to a bad token.
 */
export async function applySession(page: Page, session: Session, { skipOnboarding = true } = {}) {
  await page.addInitScript(
    ({ token, expiresAt, tokenKey, expiryKey, onboardingKeys }) => {
      localStorage.setItem(tokenKey, token)
      localStorage.setItem(expiryKey, String(expiresAt))
      for (const k of onboardingKeys) localStorage.setItem(k, 'true')
    },
    {
      token: session.token,
      expiresAt: session.expiresAt,
      tokenKey: TOKEN_KEY,
      expiryKey: TOKEN_EXPIRY_KEY,
      onboardingKeys: skipOnboarding ? [WELCOME_SEEN_KEY, CHECKLIST_DISMISSED_KEY] : [],
    }
  )
}

export const test = base.extend<{ authedPage: Page; session: Session }>({
  session: async ({}, use, testInfo) => {
    if (!process.env.E2E_TG_BOT_TOKEN || !process.env.E2E_TG_ID) {
      testInfo.skip(true, 'E2E_TG_BOT_TOKEN / E2E_TG_ID not set — authenticated specs skipped')
      return
    }
    await use(await mintSession())
  },

  authedPage: async ({ page, session }, use) => {
    await applySession(page, session)
    await use(page)
  },
})

export { expect } from '@playwright/test'
