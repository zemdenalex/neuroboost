import { useState } from 'react'
import { useAuthContext } from '../../contexts/AuthContext'
import {
  User,
  Mail,
  Calendar,
  Award,
  Flame,
  Target,
  TrendingUp,
  Clock,
  CheckCircle,
  Edit2,
  Save,
  X,
  MessageCircle,
} from 'lucide-react'

export default function Profile() {
  const { user, updateProfile } = useAuthContext()
  const [isEditingName, setIsEditingName] = useState(false)
  const [displayName, setDisplayName] = useState(user?.display_name || '')
  const [saving, setSaving] = useState(false)

  // Mock gamification stats (will be real when backend supports it)
  const stats = {
    xp: 1250,
    level: 5,
    streakDays: 7,
    longestStreak: 14,
    tasksCompleted: 42,
    eventsCompleted: 28,
    reflectionsCount: 15,
    weeklyGoalProgress: 68,
  }

  // Mock badges
  const badges = [
    { id: '1', name: 'Early Bird', icon: '🌅', description: 'Complete a task before 8 AM', earned: true },
    { id: '2', name: 'Streak Starter', icon: '🔥', description: '7 day streak', earned: true },
    { id: '3', name: 'Reflector', icon: '🪞', description: 'Complete 10 reflections', earned: true },
    { id: '4', name: 'Task Master', icon: '✅', description: 'Complete 100 tasks', earned: false },
    { id: '5', name: 'Week Warrior', icon: '⚔️', description: 'Complete all weekly goals', earned: false },
    { id: '6', name: 'Night Owl', icon: '🦉', description: 'Complete a task after 10 PM', earned: false },
  ]

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

  const userName = user?.display_name || user?.email?.split('@')[0] || 'User'
  const memberSince = user?.created_at ? new Date(user.created_at).toLocaleDateString('en-US', {
    month: 'long',
    year: 'numeric'
  }) : 'Unknown'

  // Calculate level progress (XP needed for next level)
  const xpForCurrentLevel = (stats.level - 1) * 500
  const xpForNextLevel = stats.level * 500
  const levelProgress = ((stats.xp - xpForCurrentLevel) / (xpForNextLevel - xpForCurrentLevel)) * 100

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-8">
        {/* Profile Header */}
        <div className="bg-gradient-to-r from-zinc-900 via-zinc-900 to-blue-900/20 border border-zinc-800 rounded-lg p-6">
          <div className="flex items-start gap-6">
            {/* Avatar */}
            <div className="relative">
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
              {/* Level badge */}
              <div className="absolute -bottom-1 -right-1 w-8 h-8 rounded-full bg-yellow-500 flex items-center justify-center text-sm font-bold text-black border-2 border-zinc-900">
                {stats.level}
              </div>
            </div>

            {/* User info */}
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1">
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
                    <h1 className="text-2xl font-mono font-bold text-white">{userName}</h1>
                    <button
                      onClick={() => setIsEditingName(true)}
                      className="p-1 text-zinc-400 hover:text-white hover:bg-zinc-800 rounded"
                    >
                      <Edit2 className="w-4 h-4" />
                    </button>
                  </>
                )}
              </div>

              <div className="flex flex-wrap items-center gap-4 text-sm text-zinc-400">
                {user?.email && (
                  <span className="flex items-center gap-1">
                    <Mail className="w-4 h-4" />
                    {user.email}
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
                  Member since {memberSince}
                </span>
              </div>

              {/* XP Progress bar */}
              <div className="mt-4">
                <div className="flex items-center justify-between text-sm mb-1">
                  <span className="text-zinc-400">Level {stats.level}</span>
                  <span className="text-zinc-400">{stats.xp} / {xpForNextLevel} XP</span>
                </div>
                <div className="h-2 bg-zinc-800 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-blue-600 to-blue-400 transition-all duration-500"
                    style={{ width: `${levelProgress}%` }}
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatCard
            icon={<Flame className="w-6 h-6 text-orange-400" />}
            label="Current Streak"
            value={`${stats.streakDays} days`}
            subValue={`Best: ${stats.longestStreak} days`}
          />
          <StatCard
            icon={<CheckCircle className="w-6 h-6 text-green-400" />}
            label="Tasks Completed"
            value={stats.tasksCompleted.toString()}
            subValue="This month: 12"
          />
          <StatCard
            icon={<Clock className="w-6 h-6 text-blue-400" />}
            label="Events Completed"
            value={stats.eventsCompleted.toString()}
            subValue="This month: 8"
          />
          <StatCard
            icon={<Target className="w-6 h-6 text-purple-400" />}
            label="Reflections"
            value={stats.reflectionsCount.toString()}
            subValue="Avg focus: 78%"
          />
        </div>

        {/* Weekly Goal Progress */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-zinc-400" />
              <h2 className="text-lg font-mono font-semibold text-white">Weekly Progress</h2>
            </div>
            <span className="text-2xl font-bold text-blue-400">{stats.weeklyGoalProgress}%</span>
          </div>
          <div className="h-3 bg-zinc-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-green-600 to-green-400 transition-all duration-500"
              style={{ width: `${stats.weeklyGoalProgress}%` }}
            />
          </div>
          <div className="flex justify-between mt-2 text-xs text-zinc-500">
            <span>0%</span>
            <span>Goal: 80%</span>
            <span>100%</span>
          </div>
        </div>

        {/* Badges */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Award className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Badges</h2>
            <span className="text-sm text-zinc-500">
              ({badges.filter(b => b.earned).length}/{badges.length})
            </span>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {badges.map((badge) => (
              <div
                key={badge.id}
                className={`p-3 rounded-lg border transition-all ${
                  badge.earned
                    ? 'bg-zinc-800/50 border-zinc-700'
                    : 'bg-zinc-900/50 border-zinc-800 opacity-50'
                }`}
              >
                <div className="flex items-center gap-3">
                  <span className="text-2xl">{badge.icon}</span>
                  <div>
                    <p className={`font-mono text-sm ${badge.earned ? 'text-white' : 'text-zinc-500'}`}>
                      {badge.name}
                    </p>
                    <p className="text-xs text-zinc-500">{badge.description}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Activity Calendar Placeholder */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Calendar className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Activity</h2>
          </div>
          <div className="h-32 flex items-center justify-center text-zinc-500 border border-dashed border-zinc-700 rounded-lg">
            <p className="text-sm">Activity heatmap coming soon...</p>
          </div>
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