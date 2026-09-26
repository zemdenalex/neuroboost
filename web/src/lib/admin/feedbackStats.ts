/**
 * Counts for the Admin Overview (audit A2, docs/tasks-web-cleanup.md 4.1).
 * The page must pass the UNFILTERED list: the Backlog's filtered one made
 * «Total» mean whatever the filter was.
 */
export function feedbackStats<S extends string, T extends string>(
  items: ReadonlyArray<{ status: string; type: string }>,
  statuses: readonly S[],
  types: readonly T[],
): { total: number; byStatus: Record<S, number>; byType: Record<T, number> } {
  const byStatus = Object.fromEntries(statuses.map((s) => [s, items.filter((i) => i.status === s).length])) as Record<S, number>
  const byType = Object.fromEntries(types.map((t) => [t, items.filter((i) => i.type === t).length])) as Record<T, number>
  return { total: items.length, byStatus, byType }
}
