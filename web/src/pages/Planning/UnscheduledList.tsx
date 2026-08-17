import type React from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, ListTodo, CheckCircle2, Plus } from 'lucide-react'
import type { PlanningTask } from '../../api/planning'
import { EmptyState } from '../../components/ui/EmptyState'
import { planningEmptyVariant } from '../../lib/planning/planningEmptyState'

interface Props {
  tasks: PlanningTask[]
  scheduledHours: number
  onDragStart: (e: React.DragEvent, task: PlanningTask) => void
}

export function UnscheduledList({ tasks, scheduledHours, onDragStart }: Props) {
  const { t } = useTranslation('planning')
  const emptyVariant = planningEmptyVariant(tasks.length, scheduledHours)

  return (
    <aside data-hint="planning.unscheduled" className="flex flex-col h-full bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
      <header className="px-4 py-3 border-b border-zinc-800 flex items-center justify-between">
        <span className="text-xs uppercase tracking-wider text-zinc-500">
          {t('unscheduledTasks')}
        </span>
        <span className="text-xs font-mono text-zinc-400 bg-zinc-800 px-2 py-0.5 rounded-full">
          {tasks.length}
        </span>
      </header>
      <ul className="flex-1 overflow-y-auto p-2 space-y-1">
        {emptyVariant === 'noTasks' ? (
          <li className="px-2">
            <EmptyState
              icon={ListTodo}
              title={t('emptyNoTasksTitle')}
              description={t('emptyNoTasksHint')}
              cta={{ to: '/tasks', label: t('goToTasks'), icon: Plus }}
            />
          </li>
        ) : emptyVariant === 'allScheduled' ? (
          <li className="px-2">
            <EmptyState
              icon={CheckCircle2}
              title={t('noUnscheduledTasks')}
              description={t('noUnscheduledTasksHint')}
            />
          </li>
        ) : (
          tasks.map((task) => (
            <li
              key={task.id}
              draggable
              onDragStart={(e) => onDragStart(e, task)}
              className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 rounded cursor-grab active:cursor-grabbing"
            >
              <div className="text-sm text-white font-mono line-clamp-2">{task.title}</div>
              {task.estimated_minutes != null && (
                <div className="flex items-center gap-1 mt-1 text-xs text-zinc-500">
                  <Clock className="w-3 h-3" />
                  <span>
                    ~{(task.estimated_minutes / 60).toFixed(1)}
                    {t('hoursShort')}
                  </span>
                </div>
              )}
            </li>
          ))
        )}
      </ul>
    </aside>
  )
}
