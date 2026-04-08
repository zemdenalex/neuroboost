import { api } from './client'

export type FeedbackType = 'bug' | 'feature' | 'design' | 'improvement' | 'idea' | 'other'
export type FeedbackStatus = 'open' | 'triaged' | 'approved' | 'denied' | 'in_progress' | 'resolved' | 'closed'
export type FeedbackPriority = 'low' | 'medium' | 'high' | 'critical'

export interface Feedback {
  id: string
  user_id?: string
  type: FeedbackType
  title: string
  description: string
  page_url?: string
  user_agent?: string
  status: FeedbackStatus
  priority: FeedbackPriority
  admin_notes: string
  tags: string[]
  source: string
  created_at: string
  resolved_at?: string
}

export interface CreateFeedbackRequest {
  type: FeedbackType
  title: string
  description: string
  page_url?: string
}

export interface ListFeedbackParams {
  type?: FeedbackType
  status?: FeedbackStatus
  priority?: FeedbackPriority
  source?: string
  search?: string
  sort_by?: 'created_at' | 'priority' | 'status' | 'type' | 'title'
  sort_dir?: 'asc' | 'desc'
}

export interface ImportItem {
  title: string
  description: string
  type: FeedbackType
  priority: FeedbackPriority
  status: FeedbackStatus
  tags: string[]
  source: string
}

export async function createFeedback(data: CreateFeedbackRequest): Promise<Feedback> {
  return api.post<Feedback>('/feedback', {
    ...data,
    page_url: data.page_url || window.location.href,
    user_agent: navigator.userAgent,
  }, false) // Allow anonymous feedback
}

export async function listFeedback(params?: ListFeedbackParams): Promise<Feedback[]> {
  const searchParams = new URLSearchParams()
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== '') {
        searchParams.set(key, value)
      }
    }
  }
  const query = searchParams.toString()
  return api.get<Feedback[]>(`/feedback${query ? '?' + query : ''}`)
}

export async function updateFeedback(
  id: string,
  data: {
    status?: FeedbackStatus
    priority?: FeedbackPriority
    admin_notes?: string
    tags?: string[]
  },
): Promise<Feedback> {
  return api.patch<Feedback>(`/feedback/${id}`, data)
}

export async function importFeedback(items: ImportItem[]): Promise<{ count: number }> {
  return api.post<{ count: number }>('/feedback/import', { items })
}
