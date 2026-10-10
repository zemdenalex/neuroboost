import { api } from './client'

/** One release of the bot's «Что нового» (GET /api/release-notes, bot/release). */
export interface ReleaseNote {
  version: string
  released: string
  ru: string
  en: string
}

export async function getReleaseNotes(): Promise<ReleaseNote[]> {
  const notes = await api.get<ReleaseNote[]>('/api/release-notes')
  return Array.isArray(notes) ? notes : []
}
