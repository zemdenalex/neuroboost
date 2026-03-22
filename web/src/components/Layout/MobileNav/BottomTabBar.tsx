import { useState, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Calendar, CheckSquare, PlusCircle, Settings, MoreHorizontal } from 'lucide-react'
import { QuickAddDialog } from './QuickAddDialog'
import { MoreMenu } from './MoreMenu'

export function BottomTabBar() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const location = useLocation()
  const [quickAddOpen, setQuickAddOpen] = useState(false)
  const [moreOpen, setMoreOpen] = useState(false)
  const quickAddRef = useRef<HTMLButtonElement>(null)
  const moreRef = useRef<HTMLButtonElement>(null)

  const tabs = [
    { path: '/calendar', label: t('nav.calendar'), icon: Calendar },
    { path: '/tasks', label: t('nav.tasks'), icon: CheckSquare },
  ]

  return (
    <div className="fixed bottom-0 left-0 right-0 md:hidden bg-zinc-900 border-t border-zinc-800 z-50">
      <nav className="flex items-center justify-around h-16 px-2">
        {/* Calendar */}
        {tabs.map(({ path, label, icon: Icon }) => {
          const isActive = location.pathname === path
          return (
            <button
              key={path}
              onClick={() => navigate(path)}
              className="flex flex-col items-center justify-center gap-0.5 flex-1 py-1"
            >
              <Icon
                className={`w-5 h-5 ${isActive ? 'text-blue-400' : 'text-zinc-500'}`}
              />
              <span
                className={`text-[10px] font-mono ${
                  isActive ? 'text-blue-400' : 'text-zinc-500'
                }`}
              >
                {label}
              </span>
            </button>
          )
        })}

        {/* Quick Add (+) — center, special styling */}
        <div className="relative flex flex-col items-center justify-center flex-1">
          <QuickAddDialog
            open={quickAddOpen}
            onClose={() => setQuickAddOpen(false)}
            anchorRef={quickAddRef}
          />
          <button
            ref={quickAddRef}
            onClick={() => {
              setMoreOpen(false)
              setQuickAddOpen((prev) => !prev)
            }}
            className="flex flex-col items-center justify-center gap-0.5 py-1"
          >
            <div className="w-10 h-10 rounded-full bg-blue-600 flex items-center justify-center -mt-3">
              <PlusCircle className="w-5 h-5 text-white" />
            </div>
            <span className="text-[10px] font-mono text-zinc-500">{t('action.add')}</span>
          </button>
        </div>

        {/* Settings */}
        <button
          onClick={() => navigate('/settings')}
          className="flex flex-col items-center justify-center gap-0.5 flex-1 py-1"
        >
          <Settings
            className={`w-5 h-5 ${
              location.pathname === '/settings' ? 'text-blue-400' : 'text-zinc-500'
            }`}
          />
          <span
            className={`text-[10px] font-mono ${
              location.pathname === '/settings' ? 'text-blue-400' : 'text-zinc-500'
            }`}
          >
            {t('nav.settings')}
          </span>
        </button>

        {/* More */}
        <div className="relative flex flex-col items-center justify-center flex-1">
          <MoreMenu
            open={moreOpen}
            onClose={() => setMoreOpen(false)}
            anchorRef={moreRef}
          />
          <button
            ref={moreRef}
            onClick={() => {
              setQuickAddOpen(false)
              setMoreOpen((prev) => !prev)
            }}
            className="flex flex-col items-center justify-center gap-0.5 py-1"
          >
            <MoreHorizontal
              className={`w-5 h-5 ${moreOpen ? 'text-blue-400' : 'text-zinc-500'}`}
            />
            <span
              className={`text-[10px] font-mono ${
                moreOpen ? 'text-blue-400' : 'text-zinc-500'
              }`}
            >
              {t('action.more')}
            </span>
          </button>
        </div>
      </nav>
    </div>
  )
}
