/**
 * Ids between anchor and target inclusive, in list order.
 *
 * `ids` must be in the order the list is rendered, so Shift+click selects what
 * the eye sees rather than what the underlying data happens to hold.
 */
export function selectRange(ids: string[], anchorId: string, targetId: string): string[] {
  const target = ids.indexOf(targetId)
  if (target === -1) return []
  const anchor = ids.indexOf(anchorId)
  // No usable anchor (first click, or the anchor row has since been filtered
  // out): fall back to selecting the clicked row alone.
  if (anchor === -1) return [targetId]
  const [from, to] = anchor <= target ? [anchor, target] : [target, anchor]
  return ids.slice(from, to + 1)
}
