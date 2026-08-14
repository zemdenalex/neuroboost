import { useTranslation } from 'react-i18next'
import { Database } from 'lucide-react'
import { api } from '../../../api/client'
import { showToast } from '../../../components/ui/Toast'

/**
 * Export and import of the user's own data.
 *
 * Step 3 of the Settings split (2026-08-14). Reports through showToast rather
 * than the page's error banner, following Calendars/CalendarsSection.tsx — the
 * banner was the last thing tying this section to the parent's state.
 *
 * Both handlers lean on api.get/api.post throwing on a non-2xx: an earlier
 * version downloaded an "undefined" blob on a failed export and reloaded the
 * page on a failed import, each looking exactly like success.
 *
 * 🔴 The "clear all" button has NO onClick and never had one. It is rendered,
 * it is styled as destructive, and pressing it does nothing at all. Left as it
 * was because removing a visible control is a product decision — but a button
 * that promises to delete everything and silently declines is worse than
 * either implementing or removing it. Flagged rather than carried over
 * silently.
 */
export function DataSection() {
  const { t } = useTranslation('settings')

  const handleExport = async () => {
    try {
      const exported = await api.get('/export')
      const blob = new Blob([JSON.stringify(exported, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `neuroboost-export-${new Date().toISOString().split('T')[0]}.json`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      showToast(t('dataManagement.exportFailed'))
    }
  }

  const handleImport = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.json'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      try {
        const text = await file.text()
        const data = JSON.parse(text) as unknown
        await api.post('/import', data)
        window.location.reload()
      } catch {
        showToast(t('dataManagement.importFailed'))
      }
    }
    input.click()
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Database className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('dataManagement.title')}</h2>
      </div>

      <div className="flex flex-wrap gap-3">
        <button
          onClick={handleExport}
          className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors"
        >
          {t('dataManagement.export')}
        </button>
        <button
          onClick={handleImport}
          className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors"
        >
          {t('dataManagement.import')}
        </button>
        {/* No handler — see the note at the top of this file. */}
        <button className="px-4 py-2 bg-red-900/30 hover:bg-red-900/50 text-red-400 font-mono text-sm rounded-lg transition-colors border border-red-800">
          {t('dataManagement.clearAll')}
        </button>
      </div>
    </section>
  )
}
