import { useEffect, useState, useCallback, useRef } from 'react'
import { Link } from 'react-router-dom'
import { useAuthContext, useRequireAdmin } from '../../contexts/AuthContext'
import {
  listFeedback,
  updateFeedback,
  importFeedback,
  type Feedback,
  type FeedbackType,
  type FeedbackStatus,
  type FeedbackPriority,
  type ListFeedbackParams,
} from '../../api/feedback'
import { getAdminHealth, getAdminLogs, type HealthResponse, type LogEntry } from '../../api/admin'
import {
  Shield,
  Users,
  Bug,
  Lightbulb,
  Palette,
  Sparkles,
  AlertTriangle,
  CheckCircle,
  Clock,
  XCircle,
  ChevronLeft,
  ChevronDown,
  ChevronUp,
  RefreshCw,
  Activity,
  Search,
  Upload,
  MessageSquarePlus,
  ArrowUpDown,
  ThumbsUp,
  ThumbsDown,
  Eye,
  X,
  Save,
  Loader2,
  Heart,
  ScrollText,
  Database,
  Cpu,
  MemoryStick,
  Pause,
  Play,
} from 'lucide-react'

const STATUS_CONFIG: Record<FeedbackStatus, { icon: typeof Bug; color: string; label: string }> = {
  open: { icon: AlertTriangle, color: 'text-yellow-500', label: 'Open' },
  triaged: { icon: Eye, color: 'text-blue-400', label: 'Triaged' },
  approved: { icon: ThumbsUp, color: 'text-green-400', label: 'Approved' },
  denied: { icon: ThumbsDown, color: 'text-red-400', label: 'Denied' },
  in_progress: { icon: Clock, color: 'text-blue-500', label: 'In Progress' },
  resolved: { icon: CheckCircle, color: 'text-green-500', label: 'Resolved' },
  closed: { icon: XCircle, color: 'text-zinc-500', label: 'Closed' },
}

const TYPE_CONFIG: Record<FeedbackType, { icon: typeof Bug; color: string }> = {
  bug: { icon: Bug, color: 'text-red-500' },
  feature: { icon: Lightbulb, color: 'text-yellow-400' },
  design: { icon: Palette, color: 'text-purple-400' },
  improvement: { icon: Lightbulb, color: 'text-orange-400' },
  idea: { icon: Sparkles, color: 'text-emerald-400' },
  other: { icon: MessageSquarePlus, color: 'text-blue-400' },
}

const PRIORITY_CONFIG: Record<FeedbackPriority, { color: string; bg: string }> = {
  critical: { color: 'text-red-400', bg: 'bg-red-500/10' },
  high: { color: 'text-orange-400', bg: 'bg-orange-500/10' },
  medium: { color: 'text-yellow-400', bg: 'bg-yellow-500/10' },
  low: { color: 'text-zinc-400', bg: 'bg-zinc-500/10' },
}

export function Admin() {
  const { user } = useAuthContext()
  const { isAdmin, loading } = useRequireAdmin()

  const [feedback, setFeedback] = useState<Feedback[]>([])
  const [feedbackLoading, setFeedbackLoading] = useState(true)
  const [selectedTab, setSelectedTab] = useState<'overview' | 'backlog' | 'health' | 'logs' | 'users'>('backlog')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // Filters
  const [filterType, setFilterType] = useState<FeedbackType | ''>('')
  const [filterStatus, setFilterStatus] = useState<FeedbackStatus | ''>('')
  const [filterPriority, setFilterPriority] = useState<FeedbackPriority | ''>('')
  const [searchText, setSearchText] = useState('')
  const [sortBy, setSortBy] = useState<ListFeedbackParams['sort_by']>('created_at')
  const [sortDir, setSortDir] = useState<ListFeedbackParams['sort_dir']>('desc')

  // Import state
  const [importing, setImporting] = useState(false)
  const [importDone, setImportDone] = useState(false)

  const loadFeedback = useCallback(async () => {
    setFeedbackLoading(true)
    try {
      const params: ListFeedbackParams = { sort_by: sortBy, sort_dir: sortDir }
      if (filterType) params.type = filterType
      if (filterStatus) params.status = filterStatus
      if (filterPriority) params.priority = filterPriority
      if (searchText) params.search = searchText
      const data = await listFeedback(params)
      setFeedback(data)
    } catch (err) {
      console.error('Failed to load feedback:', err)
    } finally {
      setFeedbackLoading(false)
    }
  }, [filterType, filterStatus, filterPriority, searchText, sortBy, sortDir])

  useEffect(() => {
    if (isAdmin) loadFeedback()
  }, [isAdmin, loadFeedback])

  const handleStatusChange = async (id: string, status: FeedbackStatus) => {
    try {
      const updated = await updateFeedback(id, { status })
      setFeedback((prev) => prev.map((f) => (f.id === id ? updated : f)))
    } catch (err) {
      console.error('Failed to update status:', err)
    }
  }

  const handlePriorityChange = async (id: string, priority: FeedbackPriority) => {
    try {
      const updated = await updateFeedback(id, { priority })
      setFeedback((prev) => prev.map((f) => (f.id === id ? updated : f)))
    } catch (err) {
      console.error('Failed to update priority:', err)
    }
  }

  const toggleSort = (col: NonNullable<ListFeedbackParams['sort_by']>) => {
    if (sortBy === col) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortBy(col)
      setSortDir('asc')
    }
  }

  const handleImport = async () => {
    if (importing || importDone) return
    setImporting(true)
    try {
      const items = generateImportItems()
      const result = await importFeedback(items)
      setImportDone(true)
      alert(`Imported ${result.count} items`)
      loadFeedback()
    } catch (err) {
      console.error('Import failed:', err)
      alert('Import failed')
    } finally {
      setImporting(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
      </div>
    )
  }

  if (!isAdmin) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center p-4">
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-8 max-w-md text-center">
          <Shield className="w-12 h-12 text-red-500 mx-auto mb-4" />
          <h1 className="text-xl font-mono font-bold text-white mb-2">Access Denied</h1>
          <p className="text-zinc-400 mb-6">You don&apos;t have permission to access the admin panel.</p>
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
  const statsByStatus = Object.fromEntries(
    (Object.keys(STATUS_CONFIG) as FeedbackStatus[]).map((s) => [
      s,
      feedback.filter((f) => f.status === s).length,
    ]),
  ) as Record<FeedbackStatus, number>

  const statsByType = Object.fromEntries(
    (Object.keys(TYPE_CONFIG) as FeedbackType[]).map((t) => [
      t,
      feedback.filter((f) => f.type === t).length,
    ]),
  ) as Record<FeedbackType, number>

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
            { id: 'overview' as const, label: 'Overview', icon: Activity },
            { id: 'backlog' as const, label: 'Backlog', icon: Bug },
            { id: 'health' as const, label: 'Health', icon: Heart },
            { id: 'logs' as const, label: 'Logs', icon: ScrollText },
            { id: 'users' as const, label: 'Users', icon: Users },
          ].map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setSelectedTab(id)}
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
          <div className="space-y-6">
            {/* Status breakdown */}
            <div>
              <h2 className="text-sm font-mono text-zinc-400 mb-3 uppercase tracking-wide">By Status</h2>
              <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
                {(Object.keys(STATUS_CONFIG) as FeedbackStatus[]).map((s) => {
                  const cfg = STATUS_CONFIG[s]
                  const Icon = cfg.icon
                  return (
                    <div key={s} className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
                      <div className="flex items-center gap-2 mb-1">
                        <Icon className={`w-4 h-4 ${cfg.color}`} />
                        <span className="text-xs text-zinc-400">{cfg.label}</span>
                      </div>
                      <p className="text-2xl font-mono font-bold text-white">{statsByStatus[s]}</p>
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Type breakdown */}
            <div>
              <h2 className="text-sm font-mono text-zinc-400 mb-3 uppercase tracking-wide">By Type</h2>
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
                {(Object.keys(TYPE_CONFIG) as FeedbackType[]).map((t) => {
                  const cfg = TYPE_CONFIG[t]
                  const Icon = cfg.icon
                  return (
                    <div key={t} className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
                      <div className="flex items-center gap-2 mb-1">
                        <Icon className={`w-4 h-4 ${cfg.color}`} />
                        <span className="text-xs text-zinc-400 capitalize">{t}</span>
                      </div>
                      <p className="text-2xl font-mono font-bold text-white">{statsByType[t]}</p>
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Total */}
            <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
              <p className="text-zinc-400 text-sm">Total Backlog Items</p>
              <p className="text-4xl font-mono font-bold text-white mt-1">{feedback.length}</p>
            </div>
          </div>
        )}

        {/* Backlog Tab */}
        {selectedTab === 'backlog' && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-lg">
            {/* Toolbar */}
            <div className="flex flex-wrap items-center gap-3 p-4 border-b border-zinc-800">
              {/* Search */}
              <div className="relative flex-1 min-w-[200px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
                <input
                  type="text"
                  placeholder="Search titles..."
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  className="w-full pl-9 pr-3 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-sm text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>

              {/* Filters */}
              <select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value as FeedbackType | '')}
                className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1.5 font-mono"
              >
                <option value="">All Types</option>
                {(Object.keys(TYPE_CONFIG) as FeedbackType[]).map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>

              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value as FeedbackStatus | '')}
                className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1.5 font-mono"
              >
                <option value="">All Statuses</option>
                {(Object.keys(STATUS_CONFIG) as FeedbackStatus[]).map((s) => (
                  <option key={s} value={s}>{STATUS_CONFIG[s].label}</option>
                ))}
              </select>

              <select
                value={filterPriority}
                onChange={(e) => setFilterPriority(e.target.value as FeedbackPriority | '')}
                className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1.5 font-mono"
              >
                <option value="">All Priorities</option>
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </select>

              {/* Actions */}
              <button
                onClick={loadFeedback}
                disabled={feedbackLoading}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-zinc-400 hover:text-white transition-colors"
              >
                <RefreshCw className={`w-4 h-4 ${feedbackLoading ? 'animate-spin' : ''}`} />
              </button>

              <button
                onClick={handleImport}
                disabled={importing || importDone}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-zinc-800 border border-zinc-700 rounded text-zinc-300 hover:text-white transition-colors disabled:opacity-50"
              >
                {importing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Upload className="w-4 h-4" />}
                {importDone ? 'Imported' : 'Import Features'}
              </button>
            </div>

            {/* Column Headers */}
            <div className="grid grid-cols-[40px_80px_1fr_100px_120px_80px_120px] gap-2 px-4 py-2 border-b border-zinc-800 text-xs text-zinc-500 font-mono uppercase">
              <span />
              <SortHeader label="Type" col="type" current={sortBy} dir={sortDir} onSort={toggleSort} />
              <SortHeader label="Title" col="title" current={sortBy} dir={sortDir} onSort={toggleSort} />
              <span>Priority</span>
              <SortHeader label="Status" col="status" current={sortBy} dir={sortDir} onSort={toggleSort} />
              <span>Source</span>
              <SortHeader label="Created" col="created_at" current={sortBy} dir={sortDir} onSort={toggleSort} />
            </div>

            {/* List */}
            {feedbackLoading ? (
              <div className="p-8 text-center text-zinc-500">Loading...</div>
            ) : feedback.length === 0 ? (
              <div className="p-8 text-center text-zinc-500">No items found</div>
            ) : (
              <div className="divide-y divide-zinc-800">
                {feedback.map((item) => (
                  <BacklogRow
                    key={item.id}
                    item={item}
                    isExpanded={expandedId === item.id}
                    onToggle={() => setExpandedId(expandedId === item.id ? null : item.id)}
                    onStatusChange={handleStatusChange}
                    onPriorityChange={handlePriorityChange}
                    onUpdate={(updated) => setFeedback((prev) => prev.map((f) => (f.id === updated.id ? updated : f)))}
                  />
                ))}
              </div>
            )}
          </div>
        )}

        {/* Health Tab */}
        {selectedTab === 'health' && <HealthTab />}

        {/* Logs Tab */}
        {selectedTab === 'logs' && <LogsTab />}

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

// Sort header button
function SortHeader({
  label,
  col,
  current,
  dir,
  onSort,
}: {
  label: string
  col: NonNullable<ListFeedbackParams['sort_by']>
  current: ListFeedbackParams['sort_by']
  dir: ListFeedbackParams['sort_dir']
  onSort: (col: NonNullable<ListFeedbackParams['sort_by']>) => void
}) {
  return (
    <button
      onClick={() => onSort(col)}
      className="flex items-center gap-1 hover:text-zinc-300 transition-colors"
    >
      {label}
      {current === col ? (
        dir === 'asc' ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />
      ) : (
        <ArrowUpDown className="w-3 h-3 opacity-30" />
      )}
    </button>
  )
}

// Expandable backlog row
function BacklogRow({
  item,
  isExpanded,
  onToggle,
  onStatusChange,
  onPriorityChange,
  onUpdate,
}: {
  item: Feedback
  isExpanded: boolean
  onToggle: () => void
  onStatusChange: (id: string, status: FeedbackStatus) => void
  onPriorityChange: (id: string, priority: FeedbackPriority) => void
  onUpdate: (item: Feedback) => void
}) {
  const [notes, setNotes] = useState(item.admin_notes)
  const [tagsInput, setTagsInput] = useState(item.tags.join(', '))
  const [saving, setSaving] = useState(false)

  const typeCfg = TYPE_CONFIG[item.type] || TYPE_CONFIG.other
  const TypeIcon = typeCfg.icon
  const priorityCfg = PRIORITY_CONFIG[item.priority] || PRIORITY_CONFIG.medium
  const statusCfg = STATUS_CONFIG[item.status] || STATUS_CONFIG.open
  const StatusIcon = statusCfg.icon

  const handleSaveDetails = async () => {
    setSaving(true)
    try {
      const tags = tagsInput
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean)
      const updated = await updateFeedback(item.id, { admin_notes: notes, tags })
      onUpdate(updated)
    } catch (err) {
      console.error('Failed to save:', err)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      {/* Row */}
      <div
        onClick={onToggle}
        className="grid grid-cols-[40px_80px_1fr_100px_120px_80px_120px] gap-2 px-4 py-3 hover:bg-zinc-800/50 cursor-pointer items-center"
      >
        <span className="text-zinc-500">
          {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </span>
        <span className="flex items-center gap-1.5">
          <TypeIcon className={`w-4 h-4 ${typeCfg.color}`} />
          <span className="text-xs text-zinc-400 capitalize">{item.type}</span>
        </span>
        <span className="font-mono text-white truncate text-sm">{item.title}</span>
        <span
          className={`text-xs font-mono px-2 py-0.5 rounded ${priorityCfg.bg} ${priorityCfg.color} capitalize`}
        >
          {item.priority}
        </span>
        <span className="flex items-center gap-1.5">
          <StatusIcon className={`w-3.5 h-3.5 ${statusCfg.color}`} />
          <span className="text-xs text-zinc-400">{statusCfg.label}</span>
        </span>
        <span className="text-xs text-zinc-500">{item.source}</span>
        <span className="text-xs text-zinc-500">{new Date(item.created_at).toLocaleDateString()}</span>
      </div>

      {/* Expanded Detail */}
      {isExpanded && (
        <div className="px-4 pb-4 pl-14 space-y-4 bg-zinc-800/20">
          {/* Description */}
          <div>
            <label className="block text-xs text-zinc-500 mb-1 uppercase">Description</label>
            <p className="text-sm text-zinc-300 whitespace-pre-wrap">
              {item.description || '(no description)'}
            </p>
          </div>

          {/* Inline editors */}
          <div className="flex flex-wrap gap-4">
            <div>
              <label className="block text-xs text-zinc-500 mb-1 uppercase">Status</label>
              <select
                value={item.status}
                onChange={(e) => {
                  e.stopPropagation()
                  onStatusChange(item.id, e.target.value as FeedbackStatus)
                }}
                className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1 font-mono"
              >
                {(Object.keys(STATUS_CONFIG) as FeedbackStatus[]).map((s) => (
                  <option key={s} value={s}>{STATUS_CONFIG[s].label}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs text-zinc-500 mb-1 uppercase">Priority</label>
              <select
                value={item.priority}
                onChange={(e) => {
                  e.stopPropagation()
                  onPriorityChange(item.id, e.target.value as FeedbackPriority)
                }}
                className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1 font-mono"
              >
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </select>
            </div>

            {item.page_url && (
              <div>
                <label className="block text-xs text-zinc-500 mb-1 uppercase">Page URL</label>
                <p className="text-xs text-zinc-400">{item.page_url}</p>
              </div>
            )}
          </div>

          {/* Tags */}
          <div>
            <label className="block text-xs text-zinc-500 mb-1 uppercase">Tags (comma-separated)</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={tagsInput}
                onChange={(e) => setTagsInput(e.target.value)}
                onClick={(e) => e.stopPropagation()}
                className="flex-1 px-3 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-sm text-white font-mono focus:outline-none focus:border-blue-500"
                placeholder="e.g. infrastructure, phase-0"
              />
            </div>
            {item.tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mt-2">
                {item.tags.map((tag) => (
                  <span
                    key={tag}
                    className="px-2 py-0.5 bg-zinc-700 text-zinc-300 text-xs rounded font-mono"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>

          {/* Admin Notes */}
          <div>
            <label className="block text-xs text-zinc-500 mb-1 uppercase">Admin Notes</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              onClick={(e) => e.stopPropagation()}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded text-sm text-white font-mono focus:outline-none focus:border-blue-500 resize-none"
              rows={3}
              placeholder="Internal notes, reasons for approval/denial..."
            />
          </div>

          {/* Save button */}
          <button
            onClick={(e) => {
              e.stopPropagation()
              handleSaveDetails()
            }}
            disabled={saving}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 text-white text-sm font-mono rounded transition-colors"
          >
            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            Save Notes & Tags
          </button>
        </div>
      )}
    </div>
  )
}

// Health tab — shows system health stats
function HealthTab() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      const data = await getAdminHealth()
      setHealth(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load health')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    const interval = setInterval(load, 30000) // refresh every 30s
    return () => clearInterval(interval)
  }, [load])

  if (loading && !health) {
    return <div className="p-8 text-center text-zinc-500">Loading health data...</div>
  }

  if (error && !health) {
    return <div className="p-8 text-center text-red-400">{error}</div>
  }

  if (!health) return null

  const formatUptime = (seconds: number) => {
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    if (d > 0) return `${d}d ${h}h ${m}m`
    if (h > 0) return `${h}h ${m}m`
    return `${m}m`
  }

  const dbOk = health.db_status === 'ok'

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-mono text-zinc-400 uppercase tracking-wide">System Health</h2>
        <button onClick={load} className="text-zinc-400 hover:text-white transition-colors">
          <RefreshCw className="w-4 h-4" />
        </button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-1">
            <Activity className="w-4 h-4 text-blue-400" />
            <span className="text-xs text-zinc-400">Version</span>
          </div>
          <p className="text-xl font-mono font-bold text-white">{health.version}</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-1">
            <Clock className="w-4 h-4 text-green-400" />
            <span className="text-xs text-zinc-400">Uptime</span>
          </div>
          <p className="text-xl font-mono font-bold text-white">{formatUptime(health.uptime_seconds)}</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-1">
            <Database className={`w-4 h-4 ${dbOk ? 'text-green-400' : 'text-red-400'}`} />
            <span className="text-xs text-zinc-400">Database</span>
          </div>
          <p className={`text-xl font-mono font-bold ${dbOk ? 'text-green-400' : 'text-red-400'}`}>
            {dbOk ? 'OK' : 'Error'}
          </p>
          <p className="text-xs text-zinc-500 mt-1">{health.db_ping_ms}ms ping</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-1">
            <Shield className="w-4 h-4 text-purple-400" />
            <span className="text-xs text-zinc-400">Migration</span>
          </div>
          <p className="text-xl font-mono font-bold text-white">v{health.migration_version}</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-1">
            <MemoryStick className="w-4 h-4 text-orange-400" />
            <span className="text-xs text-zinc-400">Memory</span>
          </div>
          <p className="text-xl font-mono font-bold text-white">{health.memory_mb} MB</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-1">
            <Cpu className="w-4 h-4 text-cyan-400" />
            <span className="text-xs text-zinc-400">Goroutines</span>
          </div>
          <p className="text-xl font-mono font-bold text-white">{health.goroutines}</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-1">
            <Cpu className="w-4 h-4 text-zinc-400" />
            <span className="text-xs text-zinc-400">Go Version</span>
          </div>
          <p className="text-xl font-mono font-bold text-white">{health.go_version.replace('go', '')}</p>
        </div>
      </div>
    </div>
  )
}

// Logs tab — shows recent structured log entries
function LogsTab() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [levelFilter, setLevelFilter] = useState('')
  const [limitFilter, setLimitFilter] = useState(100)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [expandedIdx, setExpandedIdx] = useState<number | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const load = useCallback(async () => {
    try {
      const data = await getAdminLogs({
        level: levelFilter || undefined,
        limit: limitFilter,
      })
      setLogs(data)
    } catch (err) {
      console.error('Failed to load logs:', err)
    } finally {
      setLoading(false)
    }
  }, [levelFilter, limitFilter])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(load, 10000)
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [autoRefresh, load])

  const levelColor: Record<string, string> = {
    DEBUG: 'bg-zinc-600 text-zinc-200',
    INFO: 'bg-blue-500/20 text-blue-400',
    WARN: 'bg-yellow-500/20 text-yellow-400',
    ERROR: 'bg-red-500/20 text-red-400',
  }

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-3 p-4 border-b border-zinc-800">
        <select
          value={levelFilter}
          onChange={(e) => setLevelFilter(e.target.value)}
          className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1.5 font-mono"
        >
          <option value="">All Levels</option>
          <option value="DEBUG">DEBUG</option>
          <option value="INFO">INFO</option>
          <option value="WARN">WARN</option>
          <option value="ERROR">ERROR</option>
        </select>

        <select
          value={limitFilter}
          onChange={(e) => setLimitFilter(Number(e.target.value))}
          className="bg-zinc-800 border border-zinc-700 text-zinc-300 text-sm rounded px-2 py-1.5 font-mono"
        >
          <option value={50}>50 entries</option>
          <option value={100}>100 entries</option>
          <option value={200}>200 entries</option>
          <option value={500}>500 entries</option>
        </select>

        <button
          onClick={() => setAutoRefresh(!autoRefresh)}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-sm rounded border transition-colors ${
            autoRefresh
              ? 'bg-green-500/10 border-green-500/30 text-green-400'
              : 'bg-zinc-800 border-zinc-700 text-zinc-400'
          }`}
        >
          {autoRefresh ? <Pause className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5" />}
          {autoRefresh ? 'Auto (10s)' : 'Paused'}
        </button>

        <button
          onClick={load}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-zinc-400 hover:text-white transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </button>

        <span className="text-xs text-zinc-500 ml-auto">{logs.length} entries</span>
      </div>

      {/* Column headers */}
      <div className="grid grid-cols-[160px_70px_1fr] gap-2 px-4 py-2 border-b border-zinc-800 text-xs text-zinc-500 font-mono uppercase">
        <span>Timestamp</span>
        <span>Level</span>
        <span>Message</span>
      </div>

      {/* Log entries */}
      {loading && logs.length === 0 ? (
        <div className="p-8 text-center text-zinc-500">Loading logs...</div>
      ) : logs.length === 0 ? (
        <div className="p-8 text-center text-zinc-500">No log entries</div>
      ) : (
        <div className="divide-y divide-zinc-800 max-h-[600px] overflow-y-auto">
          {logs.map((entry, idx) => (
            <div key={idx}>
              <div
                onClick={() => setExpandedIdx(expandedIdx === idx ? null : idx)}
                className="grid grid-cols-[160px_70px_1fr] gap-2 px-4 py-2 hover:bg-zinc-800/50 cursor-pointer items-center"
              >
                <span className="text-xs text-zinc-500 font-mono">
                  {new Date(entry.timestamp).toLocaleTimeString()}
                </span>
                <span className={`text-xs font-mono px-1.5 py-0.5 rounded ${levelColor[entry.level] || 'bg-zinc-700 text-zinc-300'}`}>
                  {entry.level}
                </span>
                <span className="text-sm text-zinc-300 truncate font-mono">{entry.message}</span>
              </div>
              {expandedIdx === idx && entry.fields && Object.keys(entry.fields).length > 0 && (
                <div className="px-4 pb-3 pl-[240px]">
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(entry.fields).map(([key, value]) => (
                      <span
                        key={key}
                        className="text-xs font-mono px-2 py-0.5 bg-zinc-800 rounded text-zinc-400"
                      >
                        <span className="text-zinc-500">{key}:</span> {value}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// Import data: 117 features from the feature list
function generateImportItems() {
  type Item = {
    title: string
    description: string
    type: FeedbackType
    priority: FeedbackPriority
    status: FeedbackStatus
    tags: string[]
    source: string
  }

  const items: Item[] = []
  const src = 'imported'

  function add(
    title: string,
    category: string,
    phase: string,
    priority: FeedbackPriority,
    status: FeedbackStatus,
    notes = '',
  ) {
    items.push({
      title,
      description: notes ? `${category} — ${phase}. ${notes}` : `${category} — ${phase}`,
      type: 'feature',
      priority,
      status,
      tags: [category.toLowerCase(), phase.toLowerCase().replace(/\s/g, '-')],
      source: src,
    })
  }

  // Phase 0: Infrastructure
  add('Server setup (Ubuntu 22.04)', 'Infrastructure', 'Phase 0', 'critical', 'resolved', '62.76.228.106')
  add('Docker + Docker Compose', 'Infrastructure', 'Phase 0', 'critical', 'resolved')
  add('Nginx + SSL', 'Infrastructure', 'Phase 0', 'critical', 'resolved', 'Certbot done')
  add('PostgreSQL container', 'Infrastructure', 'Phase 0', 'critical', 'resolved')
  add('Database migrations (golang-migrate)', 'Infrastructure', 'Phase 0', 'critical', 'resolved')
  add('Health check endpoint', 'Infrastructure', 'Phase 0', 'critical', 'resolved')
  add('fail2ban + SSH hardening', 'Infrastructure', 'Phase 0', 'critical', 'resolved')
  add('CI: Build & test', 'Infrastructure', 'Phase 0', 'high', 'resolved', 'GitHub Actions')
  add('CI: Auto-deploy on merge', 'Infrastructure', 'Phase 0', 'high', 'open')
  add('CI: Database backups', 'Infrastructure', 'Phase 0', 'high', 'open', 'Daily cron')
  add('Error logging', 'Infrastructure', 'Phase 0', 'high', 'open', 'Structured JSON logs')

  // Phase 0: Authentication
  add('User table with auth fields', 'Auth', 'Phase 0', 'critical', 'resolved', 'Migration 000002')
  add('Telegram Login Widget', 'Auth', 'Phase 0', 'critical', 'in_progress', 'Backend ready, need frontend')
  add('Email/Password registration', 'Auth', 'Phase 0', 'critical', 'resolved')
  add('Email/Password login', 'Auth', 'Phase 0', 'critical', 'resolved')
  add('JWT token generation', 'Auth', 'Phase 0', 'critical', 'resolved', '30-day tokens')
  add('JWT middleware (Go)', 'Auth', 'Phase 0', 'critical', 'resolved')
  add('GET /api/auth/me', 'Auth', 'Phase 0', 'critical', 'resolved')
  add('Login page (frontend)', 'Auth', 'Phase 0', 'critical', 'open')
  add('Session refresh banner', 'Auth', 'Phase 0', 'medium', 'open')
  add('Password reset (email)', 'Auth', 'Phase 0', 'high', 'open')
  add('Email verification', 'Auth', 'Phase 0', 'medium', 'open')
  add('Google OAuth', 'Auth', 'Phase 0', 'low', 'open')

  // Phase 1: Admin Panel
  add('Admin dashboard page', 'Admin', 'Phase 1', 'critical', 'open')
  add('Admin authentication', 'Admin', 'Phase 1', 'critical', 'open', 'Role-based (is_admin flag)')
  add('User management', 'Admin', 'Phase 1', 'high', 'open')
  add('Health status view', 'Admin', 'Phase 1', 'high', 'open')
  add('Feedback/bug reports view', 'Admin', 'Phase 1', 'critical', 'open')
  add('IP ban management', 'Admin', 'Phase 1', 'medium', 'open')
  add('System logs viewer', 'Admin', 'Phase 1', 'medium', 'open')
  add('Basic analytics', 'Admin', 'Phase 1', 'medium', 'open')

  // Phase 1: Feedback System
  add('Feedback button (all pages)', 'Feedback', 'Phase 1', 'critical', 'open')
  add('Bug report form', 'Feedback', 'Phase 1', 'critical', 'open')
  add('Feature suggestion form', 'Feedback', 'Phase 1', 'critical', 'open')
  add('Screenshot attachment', 'Feedback', 'Phase 1', 'medium', 'open')
  add('Auto-capture context', 'Feedback', 'Phase 1', 'high', 'open')
  add('POST /api/feedback', 'Feedback', 'Phase 1', 'critical', 'open')
  add('GET /api/admin/feedback', 'Feedback', 'Phase 1', 'critical', 'open')
  add('PATCH /api/admin/feedback/:id', 'Feedback', 'Phase 1', 'high', 'open')
  add('Feedback table (DB)', 'Feedback', 'Phase 1', 'critical', 'open')
  add('GitHub issue auto-create', 'Feedback', 'Phase 1', 'low', 'open')
  add('Telegram notification', 'Feedback', 'Phase 1', 'high', 'open')

  // Phase 2: Events
  add('GET /api/events (list)', 'Events', 'Phase 2', 'critical', 'open')
  add('POST /api/events (create)', 'Events', 'Phase 2', 'critical', 'open')
  add('PATCH /api/events/:id', 'Events', 'Phase 2', 'critical', 'open')
  add('DELETE /api/events/:id', 'Events', 'Phase 2', 'critical', 'open')
  add('PATCH /api/events/:id/move', 'Events', 'Phase 2', 'critical', 'open', 'Drag & drop')
  add('PATCH /api/events/:id/resize', 'Events', 'Phase 2', 'critical', 'open')
  add('Recurring events (rrule)', 'Events', 'Phase 2', 'high', 'open', 'iCal format')
  add('Event exceptions', 'Events', 'Phase 2', 'high', 'open')
  add('Multi-day events', 'Events', 'Phase 2', 'high', 'open')
  add('All-day events', 'Events', 'Phase 2', 'critical', 'open')
  add('Calendar layers', 'Events', 'Phase 2', 'medium', 'open', 'Work/Personal/Health')

  // Phase 2: Tasks
  add('GET /api/tasks (list)', 'Tasks', 'Phase 2', 'critical', 'open')
  add('POST /api/tasks', 'Tasks', 'Phase 2', 'critical', 'open')
  add('PATCH /api/tasks/:id', 'Tasks', 'Phase 2', 'critical', 'open')
  add('DELETE /api/tasks/:id', 'Tasks', 'Phase 2', 'critical', 'open')
  add('POST /api/tasks/:id/schedule', 'Tasks', 'Phase 2', 'critical', 'open')
  add('PATCH /api/tasks/bulk', 'Tasks', 'Phase 2', 'high', 'open')
  add('Task priorities (0-5)', 'Tasks', 'Phase 2', 'critical', 'open')
  add('Task categories', 'Tasks', 'Phase 2', 'high', 'open', 'EMERGENCY→BUFFER')
  add('Task contexts (@home, @work)', 'Tasks', 'Phase 2', 'medium', 'open')
  add('Task energy levels', 'Tasks', 'Phase 2', 'medium', 'open', '1-5 scale')
  add('Task dependencies', 'Tasks', 'Phase 2', 'low', 'open')
  add('Subtasks (parent/child)', 'Tasks', 'Phase 2', 'medium', 'open')
  add('Time windows', 'Tasks', 'Phase 2', 'low', 'open')
  add('Task aging/bumps', 'Tasks', 'Phase 2', 'low', 'open')

  // Phase 2: Reflections
  add('POST /api/reflections', 'Reflections', 'Phase 2', 'high', 'open')
  add('GET /api/reflections', 'Reflections', 'Phase 2', 'high', 'open')
  add('GET /api/stats/week', 'Reflections', 'Phase 2', 'high', 'open')
  add('GET /api/stats/adherence', 'Reflections', 'Phase 2', 'medium', 'open')
  add('Work hours tracking', 'Reflections', 'Phase 2', 'low', 'open')

  // Phase 3: Calendar Frontend
  add('WeekGrid component', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Drag to create events', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Drag to move events', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Resize events (bottom handle)', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Current time indicator', 'Frontend', 'Phase 3', 'critical', 'open')
  add('All-day section', 'Frontend', 'Phase 3', 'critical', 'open')
  add('15-minute snap grid', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Ghost preview on drag', 'Frontend', 'Phase 3', 'high', 'open')
  add('MonthView component', 'Frontend', 'Phase 3', 'medium', 'open')
  add('Week navigation', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Keyboard shortcuts', 'Frontend', 'Phase 3', 'medium', 'open')

  // Phase 3: Task Management Frontend
  add('TaskSidebar component', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Drag task to calendar', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Task list view', 'Frontend', 'Phase 3', 'high', 'open')
  add('Quick task completion', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Task filtering', 'Frontend', 'Phase 3', 'high', 'open')
  add('DeadlineTasks timeline', 'Frontend', 'Phase 3', 'medium', 'open')

  // Phase 3: Event Editor
  add('EventEditor modal', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Title, time inputs', 'Frontend', 'Phase 3', 'critical', 'open')
  add('All-day toggle', 'Frontend', 'Phase 3', 'critical', 'open')
  add('Recurrence settings', 'Frontend', 'Phase 3', 'medium', 'open')
  add('Color picker', 'Frontend', 'Phase 3', 'medium', 'open')
  add('Reflection sliders', 'Frontend', 'Phase 3', 'high', 'open')

  // Phase 3: Layout
  add('Replace MUI → Lucide icons', 'Frontend', 'Phase 3', 'critical', 'resolved')
  add('Clean URLs (not hash)', 'Frontend', 'Phase 3', 'critical', 'resolved')
  add('HorizontalHeader', 'Frontend', 'Phase 3', 'critical', 'resolved')
  add('VerticalSidebar', 'Frontend', 'Phase 3', 'high', 'resolved')
  add('Mobile responsive', 'Frontend', 'Phase 3', 'high', 'open')
  add('Dark theme (default)', 'Frontend', 'Phase 3', 'critical', 'resolved')
  add('Light theme toggle', 'Frontend', 'Phase 3', 'low', 'open')

  // Phase 4: Bot
  add('/start command', 'Bot', 'Phase 4', 'critical', 'open')
  add('/help command', 'Bot', 'Phase 4', 'critical', 'open')
  add('/tasks command', 'Bot', 'Phase 4', 'critical', 'open')
  add('/newtask wizard', 'Bot', 'Phase 4', 'high', 'open')
  add('/today command', 'Bot', 'Phase 4', 'critical', 'open')
  add('/week command', 'Bot', 'Phase 4', 'medium', 'open')
  add('/note command', 'Bot', 'Phase 4', 'high', 'open')
  add('/stats command', 'Bot', 'Phase 4', 'medium', 'open')
  add('/settings command', 'Bot', 'Phase 4', 'medium', 'open')
  add('/feedback command', 'Bot', 'Phase 4', 'high', 'open')
  add('Persistent reply keyboard', 'Bot', 'Phase 4', 'critical', 'open')
  add('Inline keyboards', 'Bot', 'Phase 4', 'high', 'open')
  add('MiniApp button', 'Bot', 'Phase 4', 'high', 'open')
  add('Event reminders', 'Bot', 'Phase 4', 'critical', 'open', '30/10/5 min before')
  add('Daily planning nudge', 'Bot', 'Phase 4', 'high', 'open')
  add('Weekly planning nudge', 'Bot', 'Phase 4', 'high', 'open')
  add('Task deadline alerts', 'Bot', 'Phase 4', 'high', 'open')
  add('Quiet hours', 'Bot', 'Phase 4', 'high', 'open')
  add('Rate limiting', 'Bot', 'Phase 4', 'high', 'open')
  add('Snooze options', 'Bot', 'Phase 4', 'medium', 'open')
  add('New feedback notification', 'Bot', 'Phase 4', 'high', 'open')

  // Phase 5: Gamification
  add('XP system', 'Gamification', 'Phase 5', 'medium', 'open')
  add('Levels', 'Gamification', 'Phase 5', 'medium', 'open')
  add('Streaks', 'Gamification', 'Phase 5', 'medium', 'open')
  add('Basic badges', 'Gamification', 'Phase 5', 'medium', 'open')
  add('Profile stats display', 'Gamification', 'Phase 5', 'medium', 'open')
  add('Leaderboard', 'Gamification', 'Phase 5', 'low', 'open')

  return items
}

export default Admin
