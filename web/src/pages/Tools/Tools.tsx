import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Timer, Columns3, Grid2x2, Clock4 } from 'lucide-react'

interface ToolCard {
  icon: React.ReactNode
  titleKey: string
  descKey: string
  href: string
  available: boolean
  color: string
}

export default function Tools() {
  const { t } = useTranslation('tools')

  const tools: ToolCard[] = [
    {
      icon: <Timer className="w-6 h-6" />,
      titleKey: 'hub.pomodoro',
      descKey: 'hub.pomodoro.desc',
      href: '/tools/pomodoro',
      available: true,
      color: 'text-red-400 bg-red-500/10 border-red-500/20 hover:border-red-500/50',
    },
    {
      icon: <Columns3 className="w-6 h-6" />,
      titleKey: 'hub.kanban',
      descKey: 'hub.kanban.desc',
      href: '/tools/kanban',
      available: true,
      color: 'text-blue-400 bg-blue-500/10 border-blue-500/20 hover:border-blue-500/50',
    },
    {
      icon: <Grid2x2 className="w-6 h-6" />,
      titleKey: 'hub.eisenhower',
      descKey: 'hub.eisenhower.desc',
      href: '/tools/eisenhower',
      available: true,
      color: 'text-yellow-400 bg-yellow-500/10 border-yellow-500/20 hover:border-yellow-500/50',
    },
    {
      icon: <Clock4 className="w-6 h-6" />,
      titleKey: 'hub.timeBlocking',
      descKey: 'hub.timeBlocking.desc',
      href: '/tools/time-blocking',
      available: true,
      color: 'text-green-400 bg-green-500/10 border-green-500/20 hover:border-green-500/50',
    },
  ]

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-zinc-100">{t('hub.title')}</h1>
        <p className="text-zinc-400 mt-1">{t('hub.subtitle')}</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {tools.map((tool) =>
          tool.available ? (
            <Link
              key={tool.titleKey}
              to={tool.href}
              className={`flex flex-col gap-3 p-5 rounded-xl border bg-zinc-900 transition-all ${tool.color}`}
            >
              <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${tool.color}`}>
                {tool.icon}
              </div>
              <div>
                <h3 className="font-semibold text-zinc-100">{t(tool.titleKey)}</h3>
                <p className="text-sm text-zinc-400 mt-0.5">{t(tool.descKey)}</p>
              </div>
            </Link>
          ) : (
            <div
              key={tool.titleKey}
              className="relative flex flex-col gap-3 p-5 rounded-xl border bg-zinc-900 border-zinc-700/50 opacity-60 cursor-not-allowed"
            >
              <div className="absolute top-3 right-3">
                <span className="text-xs bg-zinc-700 text-zinc-400 px-2 py-0.5 rounded-full">
                  {t('hub.comingSoon')}
                </span>
              </div>
              <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${tool.color}`}>
                {tool.icon}
              </div>
              <div>
                <h3 className="font-semibold text-zinc-100">{t(tool.titleKey)}</h3>
                <p className="text-sm text-zinc-400 mt-0.5">{t(tool.descKey)}</p>
              </div>
            </div>
          )
        )}
      </div>
    </div>
  )
}
