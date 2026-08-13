/**
 * Pulls a displayable message out of whatever a `catch` block received.
 *
 * Exists because five call sites wrote `catch (err: any)` purely to reach
 * `err.message` — an `any` that switched off checking for the whole block in
 * order to read one string. TypeScript types a catch binding as `unknown` for a
 * good reason: `throw` accepts any value, so `err` really can be a string, a
 * number, or null.
 *
 * `ApiError` needs no special case here: it extends Error, so it takes the
 * first branch. A caller that wants its `code` should narrow with
 * `instanceof ApiError` instead of parsing this string.
 */
export function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message) return err.message
  if (typeof err === 'string' && err) return err
  return fallback
}
