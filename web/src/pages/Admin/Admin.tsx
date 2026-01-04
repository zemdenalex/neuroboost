import { useEffect, useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuthContext, useRequireAdmin } from '../../contexts/AuthContext'
import { listFeedback, updateFeedback, type Feedback } from '../../api/feedback'
import {
  Shield,
  Users,
  Bug,
  Lightbulb,
  AlertTriangle,
  CheckCircle,
  Clock,
  XCircle,
  ChevronLeft,
  RefreshCw,
  Activity,
} from 'lucide-react'

export function Admin() {
  const navigate = useNavigate()
  const { user } = useAuthContext()
  const { isAdmin, loading } = useRequireAdmin()

  const [feedback, setFeedback] = useState<Feedback[]>([])
  const [feedbackLoading, setFeedbackLoading] = useState(true)
  const [selectedTab, setSelectedTab] = useState<'overview' | 'feedback' | 'users'>('overview')

  // Load feedback
  useEffect(() => {
    if (isAdmin) {
      loadFeedback()
    }
  }, [isAdmin])

  const loadFeedback = async () => {
    setFeedbackLoading(true)
    try {
      const data = await listFeedback()
      setFeedback(data)
    } catch (err) {
      console.error('Failed to load feedback:', err)
    } finally {
      setFeedbackLoading(false)
    }
  }

  const handleStatusChange = async (id: string, status: Feedback['status']) => {
    try {
      await updateFeedback(id, { status })
      setFeedback((prev) => prev.map((f) => (f.id === id ? { ...f, status } : f)))
    } catch (err) {
      console.error('Failed to update status:', err)
    }
  }

  // Loading state
  if (loading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
      </div>
    )
  }

  // Not admin - access denied
  if (!isAdmin) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center p-4">
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-8 max-w-md text-center">
          <Shield className="w-12 h-12 text-red-500 mx-auto mb-4" />
          <h1 className="text-xl font-mono font-bold text-white mb-2">Access Denied</h1>
          <p className="text-zinc-400 mb-6">You don't have permission to access the admin panel.</p>
          <Link
            to="/calendar"
            className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-mono rounded-lg transition-colors"
          >
            <ChevronLeft className="w-4 h-4" />
            Back to Calendar
          </Link>
        </div>
      </div>
    )
  }

  // Stats
  const openFeedback = feedback.filter((f) => f.status === 'open').length
  const bugReports = feedback.filter((f) => f.type === 'bug').length
  const featureRequests = feedback.filter((f) => f.type === 'feature').length

  const statusIcon = {
    open: <AlertTriangle className="w-4 h-4 text-yellow-500" />,
    in_progress: <Clock className="w-4 h-4 text-blue-500" />,
    resolved: <CheckCircle className="w-4 h-4 text-green-500" />,
    closed: <XCircle className="w-4 h-4 text-zinc-500" />,
  }

  return (
    <div className="min-h-screen bg-zinc-950">
      {/* Header */}
      <header className="bg-zinc-900 border-b border-zinc-800 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link to="/calendar" className="text-zinc-400 hover:text-white transition-colors">
              <ChevronLeft className="w-5 h-5" />
            </Link>
            <Shield className="w-6 h-6 text-blue-500" />
            <h1 className="text-xl font-mono font-bold text-white">Admin Panel</h1>
          </div>
          <div className="text-sm text-zinc-400">
            Logged in as <span className="text-white">{user?.email}</span>
          </div>
        </div>
      </header>

      <div className="p-6">
        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          {[
            { id: 'overview', label: 'Overview', icon: Activity },
            { id: 'feedback', label: 'Feedback', icon: Bug },
            { id: 'users', label: 'Users', icon: Users },
          ].map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setSelectedTab(id as any)}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg font-mono text-sm transition-colors ${
                selectedTab === id
                  ? 'bg-blue-600 text-white'
                  : 'bg-zinc-900 text-zinc-400 hover:text-white hover:bg-zinc-800'
              }`}
            >
              <Icon className="w-4 h-4" />
              {label}
            </button>
          ))}
        </div>

        {/* Overview Tab */}
        {selectedTab === 'overview' && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
              <div className="flex items-center gap-3 mb-2">
                <AlertTriangle className="w-5 h-5 text-yellow-500" />
                <span className="text-zinc-400 text-sm">Open Feedback</span>
              </div>
              <p className="text-3xl font-mono font-bold text-white">{openFeedback}</p>
            </div>

            <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
              <div className="flex items-center gap-3 mb-2">
                <Bug className="w-5 h-5 text-red-500" />
                <span className="text-zinc-400 text-sm">Bug Reports</span>
              </div>
              <p className="text-3xl font-mono font-bold text-white">{bugReports}</p>
            </div>

            <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
              <div className="flex items-center gap-3 mb-2">
                <Lightbulb className="w-5 h-5 text-blue-500" />
                <span className="text-zinc-400 text-sm">Feature Requests</span>
              </div>
              <p className="text-3xl font-mono font-bold text-white">{featureRequests}</p>
            </div>
          </div>
        )}

        {/* Feedback Tab */}
        {selectedTab === 'feedback' && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-lg">
            <div className="flex items-center justify-between p-4 border-b border-zinc-800">
              <h2 className="font-mono font-semibold text-white">All Feedback</h2>
              <button
                onClick={loadFeedback}
                disabled={feedbackLoading}
                className="flex items-center gap-2 px-3 py-1.5 text-sm text-zinc-400 hover:text-white transition-colors"
              >
                <RefreshCw className={`w-4 h-4 ${feedbackLoading ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>

            {feedbackLoading ? (
              <div className="p-8 text-center text-zinc-500">Loading...</div>
            ) : feedback.length === 0 ? (
              <div className="p-8 text-center text-zinc-500">No feedback yet</div>
            ) : (
              <div className="divide-y divide-zinc-800">
                {feedback.map((item) => (
                  <div key={item.id} className="p-4 hover:bg-zinc-800/50">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          {item.type === 'bug' ? (
                            <Bug className="w-4 h-4 text-red-500" />
                          ) : (
                            <Lightbulb className="w-4 h-4 text-blue-500" />
                          )}
                          <span className="font-mono text-white truncate">{item.title}</span>
                        </div>
                        <p className="text-sm text-zinc-400 line-clamp-2">{item.description}</p>
                        <div className="flex items-center gap-4 mt-2 text-xs text-zinc-500">
                          <span>{new Date(item.created_at).toLocaleDateString()}</span>
                          {item.page_url && <span>{item.page_url}</span>}
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        {statusIcon[item.status]}
                        <select
                          value={item.status}
                          onChange={(e) => handleStatusChange(item.id, e.target.value as Feedback['status'])}
                          className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1 font-mono"
                        >
                          <option value="open">Open</option>
                          <option value="in_progress">In Progress</option>
                          <option value="resolved">Resolved</option>
                          <option value="closed">Closed</option>
                        </select>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Users Tab (placeholder) */}
        {selectedTab === 'users' && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-8 text-center text-zinc-500">
            User management coming soon
          </div>
        )}
      </div>
    </div>
  )
}

export default Admin