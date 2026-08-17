import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, Link2, Trash2, UserPlus } from 'lucide-react'
import {
  listMembers,
  inviteByEmail,
  createInviteLink,
  removeMember,
  type Calendar,
  type CalendarMember,
  type ShareRole,
} from '../../api/calendars'
import { describeCalendarError } from '../../lib/calendars/errors'
import { showToast } from '../ui/Toast'

interface Props {
  calendar: Calendar
  /** The signed-in user, so "you" can be marked and self-removal offered. */
  meId: string | undefined
}

/**
 * Who can see this calendar, and how to let someone else in.
 *
 * Denis, 17.08: "их вообще можно с кем-то вместе использовать? Вроде бы кнопки
 * поделиться нет, тогда смысл теряется". He was right — until tonight a shared
 * calendar was only a calendar that was not the personal one.
 *
 * Two ways in, because they answer different questions:
 *   - by EMAIL, when the person already has an account. They get an invitation
 *     and see it in their own calendar panel. Nothing is emailed — this project
 *     has no mail sending at all, which is exactly why the in-app list exists.
 *   - by LINK, when they might not have an account yet, or when it is simply
 *     easier to paste something into a chat.
 *
 * 🔴 The link is a bearer credential: whoever opens it becomes the invitee, and
 * we cannot know who that was. Two hours and one use are the entire defence, so
 * the hint below says so in plain words rather than hiding it in a tooltip.
 */
export function CalendarShare({ calendar, meId }: Props) {
  const { t } = useTranslation('settings')
  const [members, setMembers] = useState<CalendarMember[] | null>(null)
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<ShareRole>('editor')
  const [busy, setBusy] = useState(false)
  const [link, setLink] = useState<string | null>(null)

  const isOwner = calendar.role === 'owner'

  const load = useCallback(() => {
    listMembers(calendar.id)
      .then(setMembers)
      .catch(() => {
        setMembers([])
        showToast(t('share.loadFailed'))
      })
  }, [calendar.id, t])

  useEffect(() => {
    void load()
  }, [load])

  const handleInvite = async () => {
    const address = email.trim()
    if (!address || busy) return
    setBusy(true)
    try {
      await inviteByEmail(calendar.id, address, role)
      setEmail('')
      load()
    } catch (err) {
      const msg = describeCalendarError(err, 'share.inviteFailed')
      showToast(t(msg.key, msg.params))
    } finally {
      setBusy(false)
    }
  }

  const handleLink = async () => {
    if (busy) return
    setBusy(true)
    try {
      const invite = await createInviteLink(calendar.id, role)
      const url = `${window.location.origin}/i/${invite.token}`
      setLink(url)
      // Copying can fail — a page without focus, or a browser that refuses the
      // permission. The link is shown on screen either way, so a failed copy is
      // an inconvenience rather than a dead end, and saying "copied" when it
      // was not would be the actual defect.
      try {
        await navigator.clipboard.writeText(url)
        showToast(t('share.linkCopied'))
      } catch {
        /* the link is rendered below; the user can select it */
      }
    } catch (err) {
      const msg = describeCalendarError(err, 'share.inviteFailed')
      showToast(t(msg.key, msg.params))
    } finally {
      setBusy(false)
    }
  }

  const handleRemove = async (userId: string) => {
    if (busy) return
    setBusy(true)
    try {
      await removeMember(calendar.id, userId)
      load()
    } catch (err) {
      const msg = describeCalendarError(err, 'share.inviteFailed')
      showToast(t(msg.key, msg.params))
    } finally {
      setBusy(false)
    }
  }

  const roleLabel = (r: CalendarMember['role']) =>
    r === 'owner' ? t('share.owner') : r === 'editor' ? t('share.editor') : t('share.viewer')

  return (
    <div data-testid="calendar-share" className="mt-2 rounded border border-zinc-800 bg-zinc-950/40 p-2">
      <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
        {t('share.members')}
      </h4>

      <ul className="space-y-1">
        {(members ?? []).map((m) => {
          const isMe = m.user_id === meId
          // The owner may remove anyone but themselves; everyone else may
          // remove only themselves, which is how leaving works.
          const canRemove = m.role !== 'owner' && (isOwner || isMe)
          return (
            <li
              key={m.user_id}
              data-testid="calendar-member"
              className="flex flex-wrap items-center gap-2 rounded px-1 py-1 text-xs hover:bg-zinc-800/60"
            >
              <span className="min-w-0 flex-1 truncate text-zinc-200" title={m.email ?? ''}>
                {m.display_name || m.email || m.user_id.slice(0, 8)}
                {isMe && <span className="ml-1 text-zinc-500">({t('share.you')})</span>}
              </span>
              <span className="shrink-0 text-zinc-500">{roleLabel(m.role)}</span>
              {m.status === 'invited' && (
                <span className="shrink-0 rounded bg-amber-500/15 px-1 text-amber-400">
                  {t('share.pending')}
                </span>
              )}
              {canRemove && (
                <button
                  type="button"
                  data-testid="calendar-member-remove"
                  onClick={() => void handleRemove(m.user_id)}
                  disabled={busy}
                  title={isMe ? t('share.leave') : t('share.remove')}
                  className="shrink-0 rounded p-1 text-zinc-500 transition-colors hover:text-red-400 disabled:opacity-40"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              )}
            </li>
          )
        })}
      </ul>

      {isOwner && (
        <>
          {/* Role first, then the two ways to use it — both the email invite and
              the link grant whatever is selected here, so one control governs
              both rather than each carrying its own and disagreeing. */}
          <div className="mt-3 flex items-center gap-2 text-xs">
            <span className="text-zinc-500">{t('share.roleLabel')}</span>
            <select
              data-testid="share-role"
              aria-label={t('share.roleLabel')}
              value={role}
              onChange={(e) => setRole(e.target.value as ShareRole)}
              className="rounded border border-zinc-700 bg-zinc-800 px-1 py-0.5 text-white"
            >
              <option value="editor">{t('share.editor')}</option>
              <option value="viewer">{t('share.viewer')}</option>
            </select>
          </div>

          <div className="mt-2 flex gap-2">
            <input
              data-testid="share-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') void handleInvite()
              }}
              placeholder={t('share.emailPlaceholder')}
              className="min-w-0 flex-1 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-sm text-white focus:border-blue-500 focus:outline-none"
            />
            <button
              type="button"
              data-testid="share-invite"
              onClick={() => void handleInvite()}
              disabled={busy || !email.trim()}
              aria-label={t('share.inviteAria')}
              title={t('share.invite')}
              className="shrink-0 rounded bg-blue-600 px-2 py-1 text-white transition-colors hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500"
            >
              <UserPlus className="h-4 w-4" />
            </button>
          </div>

          <div className="mt-3 border-t border-zinc-800 pt-2">
            <button
              type="button"
              data-testid="share-link"
              onClick={() => void handleLink()}
              disabled={busy}
              className="flex w-full items-center justify-center gap-1 rounded border border-zinc-700 px-2 py-1 text-xs text-zinc-300 transition-colors hover:bg-zinc-800 disabled:opacity-40"
            >
              <Link2 className="h-3.5 w-3.5" />
              {t('share.createLink')}
            </button>

            {link && (
              <div className="mt-2 flex items-center gap-1">
                {/* readOnly, not disabled: a disabled input cannot be selected,
                    and selecting the text by hand is the fallback for every
                    browser that refuses the clipboard. */}
                <input
                  data-testid="share-link-value"
                  readOnly
                  value={link}
                  onFocus={(e) => e.currentTarget.select()}
                  className="min-w-0 flex-1 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 font-mono text-[11px] text-zinc-300"
                />
                <button
                  type="button"
                  onClick={() => void navigator.clipboard.writeText(link).then(() => showToast(t('share.linkCopied')))}
                  title={t('share.linkCopied')}
                  className="shrink-0 rounded p-1 text-zinc-400 hover:text-white"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
            )}

            <p className="mt-1 text-[11px] leading-snug text-zinc-500">{t('share.linkHint')}</p>
          </div>
        </>
      )}
    </div>
  )
}
