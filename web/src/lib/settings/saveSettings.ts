import type { User, UpdateUserRequest, UserSettings } from '../../api/auth'

export interface SettingsDeps {
  getMe: () => Promise<User>
  updateMe: (data: UpdateUserRequest) => Promise<User>
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
export function createSettingsSaver(deps: SettingsDeps) {
  let queue: Promise<unknown> = Promise.resolve()
  return (patch: Partial<UserSettings>): Promise<User> => {
    const run = queue.then(async () => {
      const fresh = await deps.getMe()
      const settings = { ...(fresh.settings ?? {}), ...patch }
      return deps.updateMe({ settings })
    })
    // A rejected save must not jam every later one.
    queue = run.catch(() => undefined)
    return run
  }
}
