import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  CalendarDays,
  CheckSquare,
  LayoutList,
  BarChart2,
  Timer,
  Clock,
  Brain,
  ArrowRight,
} from 'lucide-react'

interface FeatureCard {
  key: string
  icon: React.ReactNode
}

const FEATURES: FeatureCard[] = [
  { key: 'calendar', icon: <CalendarDays size={28} className="text-blue-400" /> },
  { key: 'tasks', icon: <CheckSquare size={28} className="text-green-400" /> },
  { key: 'planning', icon: <LayoutList size={28} className="text-purple-400" /> },
  { key: 'reflections', icon: <BarChart2 size={28} className="text-yellow-400" /> },
  { key: 'pomodoro', icon: <Timer size={28} className="text-red-400" /> },
  { key: 'timeBlocking', icon: <Clock size={28} className="text-cyan-400" /> },
]

export function Landing() {
  const { t } = useTranslation('home')

  return (
    <div className="min-h-screen bg-black text-zinc-100 font-mono overflow-y-auto">
      {/* Hero */}
      <section className="min-h-screen flex flex-col items-center justify-center px-6 py-20 text-center">
        <Brain size={64} className="text-blue-400 mb-6" />
        <h1 className="text-6xl md:text-8xl font-bold tracking-tight mb-4">NeuroBoost</h1>
        <p className="text-zinc-400 text-lg md:text-xl max-w-2xl mb-10 leading-relaxed">
          {t('landing.tagline')}
        </p>
        <div className="flex flex-col sm:flex-row gap-4">
          <Link
            to="/login"
            className="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-500 text-white font-semibold rounded-lg transition-colors"
          >
            {t('landing.getStarted')}
            <ArrowRight size={18} />
          </Link>
          <a
            href="#features"
            className="inline-flex items-center gap-2 px-6 py-3 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-100 font-semibold rounded-lg transition-colors"
          >
            {t('landing.learnMore')}
          </a>
        </div>
      </section>

      {/* Problem / Solution */}
      <section className="px-6 py-20 border-t border-zinc-800 max-w-5xl mx-auto w-full">
        <div className="grid md:grid-cols-2 gap-8">
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-8">
            <h3 className="text-red-400 font-semibold text-sm uppercase tracking-widest mb-3">
              {t('landing.problemTitle')}
            </h3>
            <p className="text-zinc-300 leading-relaxed">{t('landing.problem')}</p>
          </div>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-8">
            <h3 className="text-green-400 font-semibold text-sm uppercase tracking-widest mb-3">
              {t('landing.solutionTitle')}
            </h3>
            <p className="text-zinc-300 leading-relaxed">{t('landing.solution')}</p>
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="px-6 py-20 border-t border-zinc-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">{t('landing.featuresTitle')}</h2>
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {FEATURES.map(({ key, icon }) => (
              <div
                key={key}
                className="bg-zinc-900 border border-zinc-800 hover:border-zinc-600 rounded-xl p-6 transition-colors"
              >
                <div className="mb-4">{icon}</div>
                <h4 className="font-semibold text-zinc-100 mb-2">
                  {t(`landing.feature.${key}.title`)}
                </h4>
                <p className="text-zinc-400 text-sm leading-relaxed">
                  {t(`landing.feature.${key}.desc`)}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="px-6 py-20 border-t border-zinc-800 text-center">
        <div className="max-w-2xl mx-auto">
          <h2 className="text-3xl font-bold mb-4">{t('landing.ctaTitle')}</h2>
          <p className="text-zinc-400 mb-8">{t('landing.ctaText')}</p>
          <Link
            to="/login"
            className="inline-flex items-center gap-2 px-8 py-4 bg-blue-600 hover:bg-blue-500 text-white font-semibold rounded-lg text-lg transition-colors"
          >
            {t('landing.ctaButton')}
            <ArrowRight size={20} />
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="px-6 py-8 border-t border-zinc-800 text-center text-zinc-500 text-sm">
        {t('landing.footer', { year: new Date().getFullYear() })}
      </footer>
    </div>
  )
}
