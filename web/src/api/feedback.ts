import { api } from './client'

export interface Feedback {
  id: string
  user_id?: string
  type: 'bug' | 'feature' | 'other'
  title: string
  description: string
  page_url?: string
  user_agent?: string
  status: 'open' | 'in_progress' | 'resolved' | 'closed'
  priority: 'low' | 'medium' | 'high' | 'critical'
  created_at: string
  resolved_at?: string
}

export interface CreateFeedbackRequest {
  type: 'bug' | 'feature' | 'other'
  title: string
  description: string
  page_url?: string
}

export async function createFeedback(data: CreateFeedbackRequest): Promise<Feedback> {
  return api.post<Feedback>('/feedback', {
    ...data,
    page_url: data.page_url || window.location.href,
    user_agent: navigator.userAgent,
  }, false) // Allow anonymous feedback
}

export async function listFeedback(status?: string): Promise<Feedback[]> {
  const query = status ? `?status=${status}` : ''
  return api.get<Feedback[]>(`/feedback${query}`)
}

export async function updateFeedback(id: string, data: Partial<Feedback>): Promise<Feedback> {
  return api.patch<Feedback>(`/feedback/${id}`, data)
}
