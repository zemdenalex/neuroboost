/**
 * Latest request wins. Each call wraps a promise; it resolves to the value only
 * if no newer call was made meanwhile, otherwise to undefined. Paging the
 * month fast sends October, then November; if October answers last, it must
 * not repaint November's grid.
 */
export function createLatestOnly() {
  let seq = 0
  return async function latest<T>(p: Promise<T>): Promise<T | undefined> {
    const mine = ++seq
    const value = await p
    return mine === seq ? value : undefined
  }
}
