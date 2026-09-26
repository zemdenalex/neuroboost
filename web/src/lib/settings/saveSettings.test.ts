import { describe, it, expect } from 'vitest'
import { createSettingsSaver } from './saveSettings'
import type { User, UpdateUserRequest } from '../../api/auth'

// A fake server: one settings blob, PATCH replaces it whole (gotcha 21).
function fakeServer(initial: Record<string, unknown>) {
  let settings = { ...initial }
  let reads = 0
  let failRead = false
  const writes: Record<string, unknown>[] = []
  return {
    deps: {
      getMe: async (): Promise<User> => {
        reads++
        await new Promise((r) => setTimeout(r, 1))
        if (failRead) throw new Error('down')
        return { settings: { ...settings } } as unknown as User
      },
      updateMe: async (data: UpdateUserRequest): Promise<User> => {
        await new Promise((r) => setTimeout(r, 1))
        settings = { ...(data.settings as Record<string, unknown>) }
        writes.push(settings)
        return { settings: { ...settings } } as unknown as User
      },
    },
    get settings() {
      return settings
    },
    get reads() {
      return reads
    },
    writes,
    failReads(on = true) {
      failRead = on
    },
  }
}

describe('createSettingsSaver', () => {
  // The bot writes day_tasks_enabled while a web tab is open: the tab's next
  // save must not put back the copy it loaded before.
  it('merges over what the server holds now, not over the tab', async () => {
    const srv = fakeServer({ ui_scale: 1, day_tasks_enabled: false, bot: { lang: 'ru' } })
    const save = createSettingsSaver(srv.deps)
    await save({ ui_scale: 1.2 })
    expect(srv.settings).toEqual({ ui_scale: 1.2, day_tasks_enabled: false, bot: { lang: 'ru' } })
  })

  it('does not write when the read fails', async () => {
    const srv = fakeServer({ ui_scale: 1 })
    srv.failReads()
    const save = createSettingsSaver(srv.deps)
    await expect(save({ ui_scale: 2 })).rejects.toThrow()
    expect(srv.writes).toHaveLength(0)
  })

  // Two saves in a row, the second started before the first finished: both
  // survive. Without the queue both would read the same old blob.
  it('keeps both of two quick saves', async () => {
    const srv = fakeServer({})
    const save = createSettingsSaver(srv.deps)
    await Promise.all([save({ work_start: '07:11' }), save({ work_end: '19:00' })])
    expect(srv.settings).toEqual({ work_start: '07:11', work_end: '19:00' })
  })

  // A rejected save must not jam the queue for every later one.
  it('carries on after a failed save', async () => {
    const srv = fakeServer({})
    const save = createSettingsSaver(srv.deps)
    srv.failReads()
    await expect(save({ work_start: '06:00' })).rejects.toThrow()
    srv.failReads(false)
    await save({ work_end: '18:00' })
    expect(srv.settings).toEqual({ work_end: '18:00' })
  })
})

// The web's language is the account's locale only; the bot keeps its own
// settings.bot.lang (Denis 26.09). A write here must not touch the blob.
describe('language write', () => {
  it('sends the locale alone and leaves the bot language as it is', async () => {
    const sent: UpdateUserRequest[] = []
    const save = createSettingsSaver({
      getMe: async () => ({ settings: { bot: { lang: 'ru' } } }) as unknown as User,
      updateMe: async (data) => { sent.push(data); return {} as User },
    })
    await save.language('en')
    expect(sent).toEqual([{ locale: 'en' }])
  })
})

// Gap list row 16: the priority style is the bot's key, written from the web.
describe('bot setting write', () => {
  it('sets one bot key and keeps the rest of the bot section and the blob', async () => {
    const sent: UpdateUserRequest[] = []
    const save = createSettingsSaver({
      getMe: async () => ({ settings: { work_start: '09:00', bot: { lang: 'en', keywords: { созвон: {} } } } }) as unknown as User,
      updateMe: async (data) => { sent.push(data); return {} as User },
    })
    await save.botSetting('priority_style', 'dot')
    expect(sent[0].settings).toEqual({ work_start: '09:00', bot: { lang: 'en', keywords: { созвон: {} }, priority_style: 'dot' } })
    expect(sent[0].locale).toBeUndefined()
  })
})
