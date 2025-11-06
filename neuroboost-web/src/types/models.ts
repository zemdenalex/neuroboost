export interface Event { id: string; title: string; startsAt: string; endsAt: string; allDay?: boolean }
export interface Task { id: string; title: string; priority: number; status: 'OPEN'|'IN_PROGRESS'|'DONE'|'CANCELLED' }
export interface Opportunity { id: string; title: string; status: string }
export interface Need { id: string; title: string; satisfied: boolean }
export type NodeKind = 'DREAM'|'GOAL'|'PROJECT'|'PHASE'|'TASK'|'EVENT'|'OPPORTUNITY'|'NEED'
