/**
 * Where to send someone after they sign in, when a `?next=` was carried along.
 *
 * Added 17.08.2026 for calendar invite links: an unauthenticated visitor
 * opening /i/<token> is sent to login with `next=/i/<token>`, and without this
 * they would land on /home with the invitation silently dropped. The token is
 * single-use and lives two hours, so "silently dropped" means "ask for a new
 * link".
 *
 * 🔴 A redirect target taken from the URL is an open-redirect hole if it is
 * used as given. `next=https://evil.example/login` on a link that looks like
 * ours would bounce the user to a copy of our login page, wearing our domain in
 * the referrer. So only a same-site absolute PATH is accepted:
 *
 *   - must start with a single `/`  → rules out `https://…` and `//evil.host`,
 *     which browsers treat as protocol-relative and therefore as another origin;
 *   - `\` is rejected too — some browsers normalise `/\evil.host` to `//evil.host`;
 *   - anything else falls back to the caller's default rather than throwing.
 */
export function safeNextPath(raw: string | null | undefined, fallback = '/home'): string {
  if (typeof raw !== 'string' || raw === '') return fallback

  // A `next` written by our own code is encoded once; a browser may have
  // decoded it already. Decoding a value that needs no decoding is harmless,
  // and failing to decode one that does would reject a valid path.
  let value: string
  try {
    value = decodeURIComponent(raw)
  } catch {
    return fallback
  }

  if (!value.startsWith('/')) return fallback
  if (value.startsWith('//') || value.startsWith('/\\')) return fallback
  return value
}
