import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthContext } from '../../contexts/AuthContext'
import { Layout } from '../../components/Layout'
import {
  User,
  Clock,
  Globe,
  ToggleLeft,
  Database,
  LogOut,
  AlertTriangle,
  Save,
  Loader2,
} from 'lucide-react'

export default function Settings() {
  const navigate = useNavigate()
  const { user, logout } = useAuthContext()
  const [saving, setSaving] = useState(false)
  const [showConfirmLogout, setShowConfirmLogout] = useState(false)

  // Settings state (these would be loaded from API/localStorage)
  const [headerStyle, setHeaderStyle] = useState<'horizontal' | 'vertical'>('horizontal')
  const [timezone, setTimezone] = useState(user?.timezone || 'Europe/Moscow')
  const [workDays, setWorkDays] = useState(['Mon', 'Tue', 'Wed', 'Thu', 'Fri'])
  const [workStart, setWorkStart] = useState('09:00')
  const [workEnd, setWorkEnd] = useState('17:00')

  // Feature toggles
  const [features, setFeatures] = useState({
    dreams: false,
    goals: false,
    projects: false,
    opportunities: false,
    needs: false,
    graph: false,
    timeline: false,
    tools: true,
  })

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const handleSave = async () => {
    setSaving(true)
    // TODO: Implement save to API
    await new Promise((r) => setTimeout(r, 500))
    setSaving(false)
  }

  const toggleFeature = (key: keyof typeof features) => {
    setFeatures((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const displayName = user?.display_name || user?.email?.split('@')[0] || 'User'

  return (
    <Layout>
      <div className="max-w-3xl mx-auto p-6 space-y-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-mono font-bold text-white">Settings</h1>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 text-white font-mono text-sm rounded-lg transition-colors"
          >
            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            Save Changes
          </button>
        </div>

        {/* Account Section */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <User className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Account</h2>
          </div>

          <div className="space-y-4">
            {/* User info display */}
            <div className="flex items-center gap-4 p-3 bg-zinc-800/50 rounded-lg">
              {user?.tg_photo_url ? (
                <img src={user.tg_photo_url} alt={displayName} className="w-12 h-12 rounded-full" />
              ) : (
                <div className="w-12 h-12 rounded-full bg-blue-600 flex items-center justify-center text-lg font-mono text-white">
                  {displayName.slice(0, 2).toUpperCase()}
                </div>
              )}
              <div>
                <p className="text-white font-mono">{displayName}</p>
                <p className="text-sm text-zinc-500">{user?.email}</p>
                {user?.tg_username && <p className="text-sm text-zinc-500">@{user.tg_username}</p>}
              </div>
            </div>

            {/* Logout */}
            <div className="pt-2 border-t border-zinc-800">
              {showConfirmLogout ? (
                <div className="flex items-center gap-3 p-3 bg-red-900/20 border border-red-800 rounded-lg">
                  <AlertTriangle className="w-5 h-5 text-red-400" />
                  <span className="flex-1 text-sm text-red-400">Are you sure you want to sign out?</span>
                  <button
                    onClick={handleLogout}
                    className="px-3 py-1 bg-red-600 hover:bg-red-700 text-white text-sm font-mono rounded transition-colors"
                  >
                    Yes, sign out
                  </button>
                  <button
                    onClick={() => setShowConfirmLogout(false)}
                    className="px-3 py-1 bg-zinc-700 hover:bg-zinc-600 text-white text-sm font-mono rounded transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              ) : (
                <button
                  onClick={() => setShowConfirmLogout(true)}
                  className="flex items-center gap-2 px-3 py-2 text-red-400 hover:bg-zinc-800 rounded-lg transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                  Sign out
                </button>
              )}
            </div>
          </div>
        </section>

        {/* Header Style */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <ToggleLeft className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Layout</h2>
          </div>

          <div className="flex gap-3">
            <button
              onClick={() => setHeaderStyle('horizontal')}
              className={`flex-1 p-3 rounded-lg border text-sm font-mono transition-colors ${
                headerStyle === 'horizontal'
                  ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                  : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
              }`}
            >
              Horizontal Top Bar
            </button>
            <button
              onClick={() => setHeaderStyle('vertical')}
              className={`flex-1 p-3 rounded-lg border text-sm font-mono transition-colors ${
                headerStyle === 'vertical'
                  ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                  : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
              }`}
            >
              Vertical Sidebar
            </button>
          </div>
        </section>

        {/* Work Hours */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Clock className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Work Hours</h2>
          </div>

          <div className="space-y-4">
            {/* Working days */}
            <div>
              <label className="block text-sm text-zinc-400 mb-2">Working Days</label>
              <div className="flex gap-2">
                {['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((day) => (
                  <button
                    key={day}
                    onClick={() =>
                      setWorkDays((prev) =>
                        prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]
                      )
                    }
                    className={`px-3 py-1.5 rounded text-sm font-mono transition-colors ${
                      workDays.includes(day)
                        ? 'bg-blue-600 text-white'
                        : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
                    }`}
                  >
                    {day}
                  </button>
                ))}
              </div>
            </div>

            {/* Time range */}
            <div className="flex gap-4">
              <div className="flex-1">
                <label className="block text-sm text-zinc-400 mb-1">Start</label>
                <input
                  type="time"
                  value={workStart}
                  onChange={(e) => setWorkStart(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="flex-1">
                <label className="block text-sm text-zinc-400 mb-1">End</label>
                <input
                  type="time"
                  value={workEnd}
                  onChange={(e) => setWorkEnd(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>
          </div>
        </section>

        {/* Timezone */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Globe className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Regional</h2>
          </div>

          <div>
            <label className="block text-sm text-zinc-400 mb-1">Timezone</label>
            <select
              value={timezone}
              onChange={(e) => setTimezone(e.target.value)}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
            >
              <option value="Europe/Moscow">Europe/Moscow (MSK)</option>
              <option value="Europe/London">Europe/London (GMT)</option>
              <option value="America/New_York">America/New_York (EST)</option>
              <option value="America/Los_Angeles">America/Los_Angeles (PST)</option>
              <option value="Asia/Tokyo">Asia/Tokyo (JST)</option>
              <option value="UTC">UTC</option>
            </select>
          </div>
        </section>

        {/* Feature Toggles */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <ToggleLeft className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Feature Toggles</h2>
          </div>

          <div className="grid grid-cols-2 gap-3">
            {Object.entries(features).map(([key, enabled]) => (
              <label
                key={key}
                className="flex items-center gap-3 p-3 bg-zinc-800/50 rounded-lg cursor-pointer hover:bg-zinc-800 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={enabled}
                  onChange={() => toggleFeature(key as keyof typeof features)}
                  className="w-4 h-4 accent-blue-600"
                />
                <span className="text-sm text-zinc-300 capitalize">{key} View</span>
              </label>
            ))}
          </div>
        </section>

        {/* Data Management */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Database className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">Data Management</h2>
          </div>

          <div className="flex flex-wrap gap-3">
            <button className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors">
              Export Data
            </button>
            <button className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors">
              Import Data
            </button>
            <button className="px-4 py-2 bg-red-900/30 hover:bg-red-900/50 text-red-400 font-mono text-sm rounded-lg transition-colors border border-red-800">
              Clear All Data
            </button>
          </div>
        </section>
      </div>
    </Layout>
  )
}