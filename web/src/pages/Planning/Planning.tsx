import { useState } from 'react'

type PlanningView = 'dreams' | 'goals' | 'projects' | 'opportunities' | 'needs' | 'graph' | 'timeline'

const VIEWS: { key: PlanningView; label: string }[] = [
  { key: 'dreams', label: 'Dreams' },
  { key: 'goals', label: 'Goals' },
  { key: 'projects', label: 'Projects' },
  { key: 'opportunities', label: 'Opportunities' },
  { key: 'needs', label: 'Needs' },
  { key: 'graph', label: 'Graph' },
  { key: 'timeline', label: 'Timeline' },
]

export default function Planning() {
  const [activeView, setActiveView] = useState<PlanningView>('dreams')

  return (
    <div className="flex flex-col h-full">
      <div className="flex gap-2 p-4 border-b border-zinc-700 overflow-x-auto">
        {VIEWS.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setActiveView(key)}
            className={`px-3 py-1.5 rounded-lg text-sm font-mono whitespace-nowrap transition-colors ${
              activeView === key
                ? 'bg-blue-600 text-white'
                : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
            }`}
          >
            {label}
          </button>
        ))}
      </div>
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="text-center">
          <p className="text-zinc-400 font-mono text-lg capitalize">{activeView} View</p>
          <p className="text-zinc-600 text-sm mt-2">Coming soon</p>
        </div>
      </div>
    </div>
  )
}
