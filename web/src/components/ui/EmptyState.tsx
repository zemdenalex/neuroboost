import type { LucideIcon } from 'lucide-react'
import { Link } from 'react-router-dom'

export interface EmptyStateCta {
  to: string
  label: string
  icon?: LucideIcon
}

/**
 * Reusable empty-state block (icon + title + optional hint + optional CTA),
 * matching the Reflections empty-state styling. Use it wherever a page or list
 * has no data, so first-time users get guidance instead of a blank space.
 */
export function EmptyState({
  icon: Icon,
  title,
  description,
  cta,
  className = 'py-10',
}: {
  icon: LucideIcon
  title: string
  description?: string
  cta?: EmptyStateCta
  className?: string
}) {
  return (
    <div className={`flex flex-col items-center justify-center text-center ${className}`}>
      <Icon className="mb-3 h-12 w-12 text-zinc-700" />
      <h3 className="mb-1 font-mono text-base font-medium text-zinc-300">{title}</h3>
      {description && <p className="mb-4 max-w-xs text-sm text-zinc-500">{description}</p>}
      {cta && (
        <Link
          to={cta.to}
          className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 font-mono text-sm text-white transition-colors hover:bg-blue-700"
        >
          {cta.icon && <cta.icon className="h-4 w-4" />}
          {cta.label}
        </Link>
      )}
    </div>
  )
}

export default EmptyState
