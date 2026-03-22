import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquarePlus, Bug, Lightbulb, Palette, Sparkles, X, Send, Loader2, CheckCircle } from 'lucide-react'
import { createFeedback, type FeedbackType } from '../../api/feedback'

interface FeedbackButtonProps {
  position?: 'bottom-right' | 'bottom-left'
}

export function FeedbackButton({ position = 'bottom-right' }: FeedbackButtonProps) {
  const { t } = useTranslation('feedback')
  const [isOpen, setIsOpen] = useState(false)
  const [type, setType] = useState<FeedbackType>('bug')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const positionClasses = position === 'bottom-right' ? 'right-4' : 'left-4'

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim() || !description.trim()) {
      setError(t('fillRequired'))
      return
    }

    setLoading(true)
    setError(null)

    try {
      await createFeedback({
        type,
        title: title.trim(),
        description: description.trim(),
        page_url: window.location.href,
      })
      setSuccess(true)
      setTimeout(() => {
        setIsOpen(false)
        setSuccess(false)
        setTitle('')
        setDescription('')
        setType('bug')
      }, 2000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('submitError'))
    } finally {
      setLoading(false)
    }
  }

  const typeConfig: Record<FeedbackType, { icon: typeof Bug; label: string; color: string }> = {
    bug: { icon: Bug, label: t('type.bug'), color: 'text-red-400' },
    feature: { icon: Lightbulb, label: t('type.feature'), color: 'text-yellow-400' },
    design: { icon: Palette, label: t('type.design'), color: 'text-purple-400' },
    idea: { icon: Sparkles, label: t('type.idea'), color: 'text-emerald-400' },
    improvement: { icon: Lightbulb, label: t('type.improvement'), color: 'text-orange-400' },
    other: { icon: MessageSquarePlus, label: t('type.other'), color: 'text-blue-400' },
  }

  return (
    <>
      {/* Floating Button */}
      <button
        onClick={() => setIsOpen(true)}
        className={`fixed bottom-4 ${positionClasses} z-40 p-3 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-full shadow-lg transition-all hover:scale-105`}
        title={t('sendFeedback')}
      >
        <MessageSquarePlus className="w-5 h-5 text-zinc-300" />
      </button>

      {/* Modal Overlay */}
      {isOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl">
            {/* Header */}
            <div className="flex items-center justify-between p-4 border-b border-zinc-800">
              <h2 className="text-lg font-mono text-white">{t('modalTitle')}</h2>
              <button
                onClick={() => setIsOpen(false)}
                className="p-1 text-zinc-400 hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Content */}
            {success ? (
              <div className="p-8 text-center">
                <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
                <p className="text-white font-mono">{t('thankYou')}</p>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="p-4 space-y-4">
                {/* Type Selection */}
                <div>
                  <label className="block text-sm text-zinc-400 mb-2">{t('typeLabel')}</label>
                  <div className="grid grid-cols-3 gap-2">
                    {(Object.keys(typeConfig) as FeedbackType[]).map((t) => {
                      const config = typeConfig[t]
                      const Icon = config.icon
                      const isSelected = type === t
                      return (
                        <button
                          key={t}
                          type="button"
                          onClick={() => setType(t)}
                          className={`flex items-center justify-center gap-1.5 py-2 px-2 rounded-lg border transition-colors ${
                            isSelected
                              ? 'bg-zinc-800 border-zinc-600'
                              : 'bg-zinc-900 border-zinc-800 hover:border-zinc-700'
                          }`}
                        >
                          <Icon className={`w-4 h-4 ${isSelected ? config.color : 'text-zinc-500'}`} />
                          <span className={`text-sm font-mono ${isSelected ? 'text-white' : 'text-zinc-500'}`}>
                            {config.label.split(' ')[0]}
                          </span>
                        </button>
                      )
                    })}
                  </div>
                </div>

                {/* Title */}
                <div>
                  <label className="block text-sm text-zinc-400 mb-1">{t('titleLabel')}</label>
                  <input
                    type="text"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                    placeholder={type === 'bug' ? t('titlePlaceholder.bug') : t('titlePlaceholder.default')}
                    maxLength={100}
                  />
                </div>

                {/* Description */}
                <div>
                  <label className="block text-sm text-zinc-400 mb-1">{t('descriptionLabel')}</label>
                  <textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500 resize-none"
                    rows={4}
                    placeholder={
                      type === 'bug'
                        ? t('descriptionPlaceholder.bug')
                        : t('descriptionPlaceholder.default')
                    }
                    maxLength={2000}
                  />
                  <p className="text-xs text-zinc-500 mt-1">{description.length}/2000</p>
                </div>

                {/* Error */}
                {error && (
                  <p className="text-sm text-red-400">{error}</p>
                )}

                {/* Submit */}
                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 disabled:cursor-not-allowed text-white font-mono rounded-lg flex items-center justify-center gap-2 transition-colors"
                >
                  {loading ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Send className="w-4 h-4" />
                  )}
                  {t('submitFeedback')}
                </button>

                {/* Info */}
                <p className="text-xs text-zinc-500 text-center">
                  {t('autoCapture')}
                </p>
              </form>
            )}
          </div>
        </div>
      )}
    </>
  )
}

export default FeedbackButton
