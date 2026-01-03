export function addDays(d: Date, n: number) { const x = new Date(d); x.setDate(x.getDate()+n); return x }
export function startOfWeek(d: Date) { const x = new Date(d); const wd = x.getDay(); x.setDate(x.getDate() - ((wd+6)%7)); return x }
