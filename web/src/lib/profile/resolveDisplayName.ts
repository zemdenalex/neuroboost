/**
 * The name to show for a user: their display name, else the local-part of their
 * email, else a (caller-supplied, localized) fallback. A whitespace-only display
 * name is treated as absent. Keeping the fallback a parameter lets the caller pass
 * a translated string instead of a hardcoded "User".
 */
export function resolveDisplayName(
  displayName: string | null | undefined,
  email: string | null | undefined,
  fallback: string,
): string {
  if (displayName && displayName.trim()) return displayName
  const localPart = email?.split('@')[0]
  if (localPart) return localPart
  return fallback
}
