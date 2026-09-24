/** The CSS colour behind each day-task square (lib/dayTasks/dayColour.ts SQUARES). */
const HEX: Record<string, string> = {
  '🟩': '#22c55e',
  '🟨': '#eab308',
  '🟧': '#f97316',
  '🟥': '#dc2626',
  '🟫': '#92400e',
  '⬛': '#3f3f46',
}

export function squareColour(square: string | undefined): string | undefined {
  return square ? HEX[square] : undefined
}
