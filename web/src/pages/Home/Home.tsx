import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function Home(){
  const { t } = useTranslation('home')

  return (
    <div className="h-full overflow-y-auto bg-black text-zinc-100 font-mono">
      {/* Hero Section */}
      <section className="min-h-screen flex flex-col items-center justify-center px-6 py-20">
        <h1 className="text-5xl font-bold">{t('title')}</h1>
        <h2 className="text-xl mt-2">{t('subtitle')}</h2>
        <p className="text-zinc-400 mt-4 max-w-xl text-center">{t('tagline')}</p>
        <div className="mt-6 flex gap-3">
          <Link className="px-4 py-2 bg-zinc-800 rounded border border-zinc-700" to="/calendar">{t('getStarted')}</Link>
          <Link className="px-4 py-2 bg-zinc-800 rounded border border-zinc-700" to="/login">{t('signIn')}</Link>
        </div>
      </section>

      {/* Problem Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">{t('problemTitle')}</h3>
        <div className="grid grid-cols-2 gap-6">
          <div className="bg-zinc-900 p-4 rounded border border-zinc-800">{t('problem')}</div>
          <div className="bg-zinc-900 p-4 rounded border border-zinc-800">{t('approach')}</div>
        </div>
      </section>

      {/* Features Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">{t('howItWorks')}</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">{t('features.timeBlocking')}</div>
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">{t('features.reminders')}</div>
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">{t('features.planVsActual')}</div>
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">{t('features.smartLearning')}</div>
        </div>
      </section>

      {/* Philosophy Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">{t('philosophy')}</h3>
        <p className="text-zinc-400 max-w-2xl">{t('philosophyText')}</p>
      </section>

      {/* CTA Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">{t('cta')}</h3>
      </section>

      {/* Footer */}
      <footer className="px-6 py-6 border-t border-zinc-800">{t('footer', { year: new Date().getFullYear() })}</footer>
    </div>
  )
}
