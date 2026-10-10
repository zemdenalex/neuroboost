import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Sparkles } from 'lucide-react'
import { getReleaseNotes, type ReleaseNote } from '../api/releaseNotes'

/**
 * «Что нового» (gap list row 19): the bot's own release notes, the same text,
 * served from bot/release so the web cannot say something the bot does not.
 */
export default function WhatsNew() {
  const { t, i18n } = useTranslation('common')
  const [notes, setNotes] = useState<ReleaseNote[] | null>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    getReleaseNotes()
      .then((n) => !cancelled && setNotes(n))
      .catch(() => !cancelled && setFailed(true))
    return () => {
      cancelled = true
    }
  }, [])

  const en = i18n.language?.startsWith('en')
  return (
    <div className="p-4 max-w-2xl mx-auto" data-testid="whats-new">
      <h1 className="text-lg font-semibold text-zinc-100 mb-4 flex items-center gap-2">
        <Sparkles size={18} className="text-blue-400" />
        {t('nav.whatsNew')}
      </h1>
      {failed && <p className="text-red-400 text-sm">{t('whatsNew.failed')}</p>}
      {!failed && notes === null && <p className="text-zinc-400 text-sm">{t('agenda.loading')}</p>}
      <div className="flex flex-col gap-4">
        {notes?.map((n) => (
          <section key={n.version} data-testid="whats-new-note" className="rounded border border-zinc-700 bg-zinc-900 p-3">
            <h2 className="font-mono text-sm text-zinc-100">
              {n.version} <span className="text-zinc-500">· {n.released}</span>
            </h2>
            <p className="mt-2 whitespace-pre-line text-sm text-zinc-300">{en ? n.en : n.ru}</p>
          </section>
        ))}
      </div>
    </div>
  )
}
