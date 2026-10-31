/**
 * The phone calendar's two choices (Denis 31.10, spec V003-20261031-arc-shapka-dobavit):
 * «header A + C choice in settings, for creation I think it should have simple
 * mode and elaborate, and simple uses A, elaborate opens like the one in
 * ticktick, month b and c depending on what is the choice of header».
 *
 * Top-level settings keys, like header_variant: createSettingsSaver merges
 * shallowly, so a nested object would lose its siblings to a second writer.
 */

/**
 * `row` (A): one line of controls; «Month» opens a real month grid.
 * `strip` (C): the week strip is the navigation; the month title opens the month.
 */
export const PHONE_HEADERS = ['row', 'strip'] as const
export type PhoneHeader = (typeof PHONE_HEADERS)[number]

/**
 * `row` for everyone until changed. Not derived from phone_month_variant =
 * strip: that would also move the e2e account (Denis's real one) into the
 * strip header and silently change every phone spec's baseline.
 */
export function readPhoneHeader(settings: { phone_header_variant?: unknown } | undefined): PhoneHeader {
  const v = settings?.phone_header_variant
  return typeof v === 'string' && (PHONE_HEADERS as readonly string[]).includes(v) ? (v as PhoneHeader) : 'row'
}

/** `simple`: one line read like the bot's. `detailed`: title, description and chips. */
export const ADD_MODES = ['simple', 'detailed'] as const
export type AddMode = (typeof ADD_MODES)[number]

export function readAddMode(settings: { add_mode?: unknown } | undefined): AddMode {
  const v = settings?.add_mode
  return typeof v === 'string' && (ADD_MODES as readonly string[]).includes(v) ? (v as AddMode) : 'simple'
}
