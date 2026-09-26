import type { User, UpdateUserRequest, UserSettings } from '../../api/auth'

export interface SettingsDeps {
  getMe: () => Promise<User>
  updateMe: (data: UpdateUserRequest) => Promise<User>
}

export interface SettingsSaver {
  (patch: Partial<UserSettings>): Promise<User>
  /**
   * The interface language, as ONE language per person (Denis 26.09): the
   * account's `locale` (web, Mini App) and `settings.bot.lang` (bot) in one
   * request, so a half-done write cannot leave them disagreeing.
   */
  language: (locale: string) => Promise<User>
}

/**
 * A settings save that reads before it writes (gotcha 21).
 *
 * PATCH /api/auth/me replaces the whole blob. Merging over the tab's copy
 * put back whatever the tab loaded: since D3 the bot writes day_tasks_* and
 * settings.bot.* too, so a tab left open reverted them on its next save.
 *
 * - The merge is over what the server holds now; a failed read writes nothing.
 * - Saves run one after another: two quick saves reading the same old blob
 *   would otherwise keep only the second.
 */
export function createSettingsSaver(deps: SettingsDeps): SettingsSaver {
  let queue: Promise<unknown> = Promise.resolve()
  const queued = (build: (fresh: User) => UpdateUserRequest): Promise<User> => {
    const run = queue.then(async () => deps.updateMe(build(await deps.getMe())))
    // A rejected save must not jam every later one.
    queue = run.catch(() => undefined)
    return run
  }

  const save = ((patch: Partial<UserSettings>) =>
    queued((fresh) => ({ settings: { ...(fresh.settings ?? {}), ...patch } }))) as SettingsSaver

  save.language = (locale: string) =>
    queued((fresh) => {
      const settings = (fresh.settings ?? {}) as Record<string, unknown>
      const bot = (settings.bot && typeof settings.bot === 'object' ? settings.bot : {}) as Record<string, unknown>
      return { locale, settings: { ...settings, bot: { ...bot, lang: locale } } as UserSettings }
    })

  return save
}
