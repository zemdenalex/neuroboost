import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useRequireAdmin } from '../../contexts/AuthContext'
import { listFeedback, updateFeedback, Feedback } from '../../api/feedback'
import {
  Shield,
  Bug,
  Lightbulb,
  MessageSquare,
  Clock,
  CheckCircle,
  XCircle,
  Loader2,
  RefreshCw,
  ChevronDown,
} from 'lucide-react'

type StatusFilter = 'all' | 'open' | 'in_progress' | 'resolved' | 'closed'

export function Admin() {
  const navigate = useNavigate()
  const { isAdmin, isAuthenticated, loading: authLoading } = useRequireAdmin()

  const [feedbackList, setFeedbackList] = useState<Feedback[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('open')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // Redirect non-admins
  useEffect(() => {
    if (!authLoading && (!isAuthenticated || !isAdmin)) {
      navigate('/login', { replace: true })
    }
  }, [authLoading, isAuthenticated, isAdmin, navigate])

  // Fetch feedback
  useEffect(() => {
    if (isAdmin) {
      fetchFeedback()
    }
  }, [isAdmin, statusFilter])

  const fetchFeedback = async () => {
    setLoading(true)
    try {
      const data = await listFeedback(statusFilter === 'all' ? undefined : statusFilter)
      setFeedbackList(data || [])
    } catch (err) {
      console.error('Failed to fetch feedback:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleStatusChange = async (id: string, newStatus: Feedback['status']) => {
    try {
      await updateFeedback(id, { status: newStatus })
      setFeedbackList((prev) =>
        prev.map((f) => (f.id === id ? { ...f, status: newStatus } : f))
      )
    } catch (err) {
      console.error('Failed to update status:', err)
    }
  }

  const handlePriorityChange = async (id: string, newPriority: Feedback['priority']) => {
    try {
      await updateFeedback(id, { priority: newPriority })
      setFeedbackList((prev) =>
        prev.map((f) => (f.id === id ? { ...f, priority: newPriority } : f))
      )
    } catch (err) {
      console.error('Failed to update priority:', err)
    }
  }

  if (authLoading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-zinc-400 animate-spin" />
      </div>
    )
  }

  if (!isAdmin) {
    return null
  }

  const typeIcons = {
    bug: Bug,
    feature: Lightbulb,
    other: MessageSquare,
  }

  const typeColors = {
    bug: 'text-red-400',
    feature: 'text-yellow-400',
    other: 'text-blue-400',
  }

  const statusColors = {
    open: 'bg-blue-900/30 text-blue-400',
    in_progress: 'bg-yellow-900/30 text-yellow-400',
    resolved: 'bg-green-900/30 text-green-400',
    closed: 'bg-zinc-800 text-zinc-400',
  }

  const priorityColors = {
    low: 'text-zinc-400',
    medium: 'text-blue-400',
    high: 'text-orange-400',
    critical: 'text-red-400',
  }

  return (
    <div className="min-h-screen bg-zinc-950 p-6">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-3">
            <Shield className="w-8 h-8 text-blue-500" />
            <h1 className="text-2xl font-mono text-white">Admin Panel</h1>
          </div>
          <button
            onClick={fetchFeedback}
            disabled={loading}
            className="flex items-center gap-2 px-3 py-2 bg-zinc-800 hover:bg-zinc-700 rounded-lg text-zinc-300 transition-colors"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4 mb-8">
          {(['open', 'in_progress', 'resolved', 'closed'] as const).map((status) => {
            const count = feedbackList.filter((f) => f.status === status).length
            return (
              <div
                key={status}
                className="p-4 bg-zinc-900 border border-zinc-800 rounded-lg"
              >
                <p className="text-sm text-zinc-400 capitalize">{status.replace('_', ' ')}</p>
                <p className="text-2xl font-mono text-white">{count}</p>
              </div>
            )
          })}
        </div>

        {/* Filters */}
        <div className="flex gap-2 mb-6">
          {(['all', 'open', 'in_progress', 'resolved', 'closed'] as const).map((status) => (
            <button
              key={status}
              onClick={() => setStatusFilter(status)}
              className={`px-3 py-1 rounded-lg font-mono text-sm transition-colors ${
                statusFilter === status
                  ? 'bg-blue-600 text-white'
                  : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
              }`}
            >
              {status === 'all' ? 'All' : status.replace('_', ' ')}
            </button>
          ))}
        </div>

        {/* Feedback List */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="w-8 h-8 text-zinc-400 animate-spin" />
          </div>
        ) : feedbackList.length === 0 ? (
          <div className="text-center py-12 text-zinc-500">
            <MessageSquare className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>No feedback found</p>
          </div>
        ) : (
          <div className="space-y-4">
            {feedbackList.map((feedback) => {
              const TypeIcon = typeIcons[feedback.type]
              const isExpanded = expandedId === feedback.id

              return (
                <div
                  key={feedback.id}
                  className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden"
                >
                  {/* Header */}
                  <div
                    className="flex items-center gap-4 p-4 cursor-pointer hover:bg-zinc-800/50"
                    onClick={() => setExpandedId(isExpanded ? null : feedback.id)}
                  >
                    <TypeIcon className={`w-5 h-5 ${typeColors[feedback.type]}`} />
                    <div className="flex-1 min-w-0">
                      <p className="text-white font-mono truncate">{feedback.title}</p>
                      <div className="flex items-center gap-2 mt-1">
                        <span className={`text-xs px-2 py-0.5 rounded ${statusColors[feedback.status]}`}>
                          {feedback.status.replace('_', ' ')}
                        </span>
                        <span className={`text-xs ${priorityColors[feedback.priority]}`}>
                          {feedback.priority}
                        </span>
                        <span className="text-xs text-zinc-500 flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {new Date(feedback.created_at).toLocaleDateString()}
                        </span>
                      </div>
                    </div>
                    <ChevronDown
                      className={`w-5 h-5 text-zinc-500 transition-transform ${
                        isExpanded ? 'rotate-180' : ''
                      }`}
                    />
                  </div>

                  {/* Expanded Content */}
                  {isExpanded && (
                    <div className="px-4 pb-4 border-t border-zinc-800">
                      <div className="mt-4 space-y-4">
                        <div>
                          <p className="text-sm text-zinc-400 mb-1">Description</p>
                          <p className="text-white whitespace-pre-wrap">{feedback.description}</p>
                        </div>

                        {feedback.page_url && (
                          <div>
                            <p className="text-sm text-zinc-400 mb-1">Page URL</p>
                            <a
                              href={feedback.page_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-blue-400 hover:underline text-sm break-all"
                            >
                              {feedback.page_url}
                            </a>
                          </div>
                        )}

                        {feedback.user_agent && (
                          <div>
                            <p className="text-sm text-zinc-400 mb-1">User Agent</p>
                            <p className="text-xs text-zinc-500 break-all">{feedback.user_agent}</p>
                          </div>
                        )}

                        {/* Actions */}
                        <div className="flex gap-4 pt-4 border-t border-zinc-800">
                          <div>
                            <p className="text-xs text-zinc-400 mb-1">Status</p>
                            <select
                              value={feedback.status}
                              onChange={(e) =>
                                handleStatusChange(feedback.id, e.target.value as Feedback['status'])
                              }
                              className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-white text-sm"
                            >
                              <option value="open">Open</option>
                              <option value="in_progress">In Progress</option>
                              <option value="resolved">Resolved</option>
                              <option value="closed">Closed</option>
                            </select>
                          </div>
                          <div>
                            <p className="text-xs text-zinc-400 mb-1">Priority</p>
                            <select
                              value={feedback.priority}
                              onChange={(e) =>
                                handlePriorityChange(feedback.id, e.target.value as Feedback['priority'])
                              }
                              className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-white text-sm"
                            >
                              <option value="low">Low</option>
                              <option value="medium">Medium</option>
                              <option value="high">High</option>
                              <option value="critical">Critical</option>
                            </select>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

export default Admin
