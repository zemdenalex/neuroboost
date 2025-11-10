export function format(date: Date) { return date.toISOString() }
export function parse(input: string) { return new Date(input) }
