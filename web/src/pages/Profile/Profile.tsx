import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthContext } from '../../contexts/AuthContext'
import { AccountLinking } from './AccountLinking'
import { dateLocale } from '../../utils/date'
import { resolveDisplayName } from '../../lib/profile/resolveDisplayName'
import { levelOf, loadAllDays, profileStats, xpOf, type ProfileStats } from '../../lib/profile/profileStats'
import { todayInZone } from '../../lib/dayTasks/dayColour'
import { listDays } from '../../api/dayTasks'
import { getTasks } from '../../api'
import { getReflections } from '../../api/reflections'
import {
  Mail,
  Calendar,
  Flame,
  Target,
  Pin,
  CheckCircle,
  Edit2,
  Save,
  X,
  MessageCircle,
} from 'lucide-react'

export default function Profile() {
  const { t, i18n } = useTranslation('profile')
  const { user, updateProfile } = useAuthContext()
  const [isEditingName, setIsEditingName] = useState(false)
  const [displayName, setDisplayName] = useState(user?.display_name || '')
  const [saving, setSaving] = useState(false)

  // Real numbers only (Denis 25.09: «real numbers, remove mock data»). Until
  // 25.09 this page showed XP 1250, level 5, a 7-day streak and badges to
  // every user, all invented. Each read fails soft: a missing source shows
  // "—" for its figures instead of a zero that looks true.
  const [stats, setStats] = useState<ProfileStats | null>(null)
  const timeZone = user?.timezone || 'Europe/Moscow'
  useEffect(() => {
    let live = true
    const today = todayInZone(new Date(), timeZone)
    void Promise.all([
      getTasks().catch(() => null),
      // All of it, not a window: XP must not shrink as old days leave it.
      loadAllDays(today, listDays).catch(() => null),
      getReflections().catch(() => null),
    ]).then(([tasks, days, reflections]) => {
      if (!live || !tasks || !days || !reflections) return
      setStats(profileStats({ today, timeZone, tasks, days, reflections: reflections.length }))
    })
    return () => {
      live = false
    }
  }, [timeZone])

  const handleSaveName = async () => {
    if (!displayName.trim()) return
    setSaving(true)
    try {
      await updateProfile({ display_name: displayName })
      setIsEditingName(false)
    } catch {
      // Error handled by context
    } finally {
      setSaving(false)
    }
  }

  const userName = resolveDisplayName(user?.display_name, user?.email, t('anonymous'))
  const memberSince = user?.created_at ? new Date(user.created_at).toLocaleDateString(dateLocale(i18n.language), {
    month: 'long',
    year: 'numeric'
  }) : t('unknownDate')

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-8">
        {/* Profile Header */}
        <div data-hint="profile.identity" className="bg-gradient-to-r from-zinc-900 via-zinc-900 to-blue-900/20 border border-zinc-800 rounded-lg p-6">
          <div className="flex items-start gap-6">
            {/* Avatar */}
            <div className="relative shrink-0">
              {user?.tg_photo_url ? (
                <img
                  src={user.tg_photo_url}
                  alt={userName}
                  className="w-24 h-24 rounded-full border-4 border-blue-600"
                />
              ) : (
                <div className="w-24 h-24 rounded-full bg-blue-600 flex items-center justify-center text-3xl font-mono text-white border-4 border-blue-500">
                  {userName.slice(0, 2).toUpperCase()}
                </div>
              )}
            </div>

            {/* User info.
                🔴 min-w-0 is load-bearing. A flex item defaults to
                min-width:auto, so it refuses to shrink below its content — and
                an email address has no break opportunities. At 375px the whole
                row grew wider than the card and the address, the XP figures and
                the progress bar ran off the right of the screen.
                Found by e2e on 17.08, and only because the run used a real
                account: the CI test account's address is short enough to fit,
                so the same spec had been passing on the same defect. */}
            <div className="flex-1 min-w-0">
              <div className="flex min-w-0 items-center gap-2 mb-1">
                {isEditingName ? (
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={displayName}
                      onChange={(e) => setDisplayName(e.target.value)}
                      className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-white font-mono text-xl focus:outline-none focus:border-blue-500"
                      autoFocus
                    />
                    <button
                      onClick={handleSaveName}
                      disabled={saving}
                      className="p-1 text-green-400 hover:bg-zinc-800 rounded"
                    >
                      <Save className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => {
                        setIsEditingName(false)
                        setDisplayName(user?.display_name || '')
                      }}
                      className="p-1 text-zinc-400 hover:bg-zinc-800 rounded"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ) : (
                  <>
                    <h1 className="truncate text-2xl font-mono font-bold text-white" title={userName}>{userName}</h1>
                    <button
                      onClick={() => setIsEditingName(true)}
                      className="shrink-0 p-1 text-zinc-400 hover:text-white hover:bg-zinc-800 rounded"
                    >
                      <Edit2 className="w-4 h-4" />
                    </button>
                  </>
                )}
              </div>

              <div className="flex flex-wrap items-center gap-4 text-sm text-zinc-400">
                {user?.email && (
                  <span className="flex min-w-0 max-w-full items-center gap-1">
                    <Mail className="w-4 h-4 shrink-0" />
                    {/* Truncated rather than wrapped: an address broken across
                        two lines is harder to read than one with an ellipsis,
                        and the full value is in the title. */}
                    <span className="truncate" title={user.email}>{user.email}</span>
                  </span>
                )}
                {user?.tg_username && (
                  <span className="flex items-center gap-1">
                    <MessageCircle className="w-4 h-4" />
                    @{user.tg_username}
                  </span>
                )}
                <span className="flex items-center gap-1">
                  <Calendar className="w-4 h-4" />
                  {t('memberSince', { date: memberSince })}
                </span>
              </div>

              {/* XP from real actions (Denis 25.09, «Day tasks drive it»):
                  +10 a done task, +25 a full day-task day, +5 a reflection. */}
              {stats && (() => {
                const xp = xpOf(stats)
                const lv = levelOf(xp)
                return (
                  <div className="mt-4" data-testid="profile-xp" title={t('real.xpRule')}>
                    <div className="flex items-center justify-between text-sm mb-1">
                      <span className="text-zinc-300">{t('level', { level: lv.level })}</span>
                      <span className="text-zinc-400 tabular-nums">{t('real.xp', { xp, into: lv.into, need: lv.need })}</span>
                    </div>
                    <div className="h-2 bg-zinc-800 rounded overflow-hidden">
                      <div className="h-full bg-blue-500" style={{ width: `${(lv.into / lv.need) * 100}%` }} />
                    </div>
                    <p className="text-xs text-zinc-500 mt-1">{t('real.xpRule')}</p>
                  </div>
                )
              })()}

            </div>
          </div>
        </div>

        {/* Linking this account to the other half of itself — the code from
            the bot, or an email for an account that arrived from Telegram.
            Placed above the stats because it is an action, and the stats are a
            report. */}
        <AccountLinking />

        {/* Stats Grid: counts over real data, "—" while loading or when a read failed */}
        <div data-testid="profile-stats" className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatCard
            icon={<Flame className="w-6 h-6 text-orange-400" />}
            label={t('real.streak')}
            value={stats ? t('streak.days', { count: stats.takenStreak }) : '—'}
            subValue={t('real.streakNote')}
          />
          <StatCard
            icon={<CheckCircle className="w-6 h-6 text-green-400" />}
            label={t('tasksCompleted')}
            value={stats ? String(stats.tasksDone) : '—'}
            subValue={stats ? t('real.thisWeek', { count: stats.tasksDoneThisWeek }) : ''}
          />
          <StatCard
            icon={<Pin className="w-6 h-6 text-blue-400" />}
            label={t('real.fullDays')}
            value={stats ? String(stats.daysFull) : '—'}
            subValue={stats ? t('real.taken', { count: stats.daysTaken }) : ''}
          />
          <StatCard
            icon={<Target className="w-6 h-6 text-purple-400" />}
            label={t('reflections')}
            value={stats ? String(stats.reflections) : '—'}
            subValue=""
          />
        </div>
    </div>
  )
}

// Stat card component
function StatCard({
  icon,
  label,
  value,
  subValue,
}: {
  icon: React.ReactNode
  label: string
  value: string
  subValue: string
}) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
      <div className="flex items-center gap-3 mb-2">
        {icon}
        <span className="text-sm text-zinc-400">{label}</span>
      </div>
      <p className="text-2xl font-bold text-white font-mono">{value}</p>
      <p className="text-xs text-zinc-500">{subValue}</p>
    </div>
  )
}