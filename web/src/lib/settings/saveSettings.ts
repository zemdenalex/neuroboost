import type { User, UpdateUserRequest, UserSettings } from '../../api/auth'

export interface SettingsDeps {
  getMe: () => Promise<User>
  updateMe: (data: UpdateUserRequest) => Promise<User>
}

export interface SettingsSaver {
  (patch: Partial<UserSettings>): Promise<User>
  /**
   * The interface language of the web and the Mini App (the account's
   * `locale`). The bot keeps its own, `settings.bot.lang` (Denis 26.09), so
   * the settings blob is neither read nor written.
   */
  language: (locale: string) => Promise<User>
  /**
   * One key of the bot section (settings.bot.*), merged into the section the
   * server holds now: it also keeps the keyword vocabulary and the language.
   */
  botSetting: (key: string, value: unknown) => Promise<User>
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
  const inTurn = (write: () => Promise<User>): Promise<User> => {
    const run = queue.then(write)
    // A rejected save must not jam every later one.
    queue = run.catch(() => undefined)
    return run
  }
  const queued = (build: (fresh: User) => UpdateUserRequest): Promise<User> =>
    inTurn(async () => deps.updateMe(build(await deps.getMe())))

  const save = ((patch: Partial<UserSettings>) =>
    queued((fresh) => ({ settings: { ...(fresh.settings ?? {}), ...patch } }))) as SettingsSaver

  const withBot = (fresh: User, key: string, value: unknown): UserSettings => {
    const settings = (fresh.settings ?? {}) as Record<string, unknown>
    const bot = (settings.bot && typeof settings.bot === 'object' ? settings.bot : {}) as Record<string, unknown>
    return { ...settings, bot: { ...bot, [key]: value } } as UserSettings
  }

  save.language = (locale: string) => inTurn(() => deps.updateMe({ locale }))

  save.botSetting = (key: string, value: unknown) =>
    queued((fresh) => ({ settings: withBot(fresh, key, value) }))

  return save
}
