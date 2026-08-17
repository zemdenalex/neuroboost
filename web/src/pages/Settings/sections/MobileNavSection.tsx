import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Smartphone } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { showToast } from '../../../components/ui/Toast'

type MobileNavType = 'bottom_tabs' | 'hamburger' | 'fab'

/**
 * Which navigation the app uses on a phone.
 *
 * Step 5 of the Settings split (2026-08-14). Three near-identical option blocks
 * became one map — the original repeated the same twenty lines three times,
 * which is how the middle one ends up with a copy-pasted key nobody notices.
 *
 * 🔴 The `isMobile` gate stays with the PARENT. It decides whether this section
 * appears at all, and moving it in here would mean mounting a component in
 * order to have it render nothing — plus every future reader would have to open
 * the file to learn the section is conditional.
 *
 * The i18n keys are derived rather than listed (`mobileNav.bottomTabs`), so the
 * option list and the strings cannot fall out of step.
 */
const OPTIONS: Array<{ value: MobileNavType; key: string }> = [
  { value: 'bottom_tabs', key: 'bottomTabs' },
  { value: 'hamburger', key: 'hamburger' },
  { value: 'fab', key: 'fab' },
]

export function MobileNavSection() {
  const { t } = useTranslation('settings')
  const { user, updateSettings } = useAuthContext()
  const [nav, setNav] = useState<MobileNavType>(user?.settings?.mobile_nav ?? 'bottom_tabs')

  useEffect(() => {
    if (user?.settings?.mobile_nav) setNav(user.settings.mobile_nav)
  // 🔴 Keyed on the account, not on the `user` OBJECT. updateSettings calls
  // setUser after every save, so a dependency on `user` re-ran this on every
  // server response — including one answering an EARLIER save, which then
  // overwrote a change the user had made in the meantime. On its own that
  // looked like a flicker; it lost the change for good as soon as the next
  // edit was built from the reverted state. Reproduced in
  // web/e2e/settings-race.spec.ts (expected 07:11, got the stored 08:00).
  //
  // Cost of the narrower key: settings changed on another device no longer
  // appear without a reload. They did not appear reliably before either — this
  // effect only ever fired on this tab's own saves.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const change = async (next: MobileNavType) => {
    setNav(next)
    try {
      await updateSettings({ mobile_nav: next })
      showToast(t('saved'))
    } catch {
      // Same neutral toast as a success — see the note in LayoutStyleSection.
      showToast(t('error.saveMobileNav'))
    }
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Smartphone className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('mobileNav.title')}</h2>
      </div>

      <div className="flex flex-col gap-3">
        {OPTIONS.map(({ value, key }) => (
          <button
            key={value}
            onClick={() => change(value)}
            className={`flex items-start gap-3 p-3 rounded-lg border text-left transition-colors ${
              nav === value
                ? 'bg-blue-600/20 border-blue-500'
                : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
            }`}
          >
            <div className="flex-1">
              <p className={`text-sm font-mono ${nav === value ? 'text-blue-400' : 'text-zinc-300'}`}>
                {t(`mobileNav.${key}`)}
              </p>
              <p className="text-xs text-zinc-500 mt-0.5">{t(`mobileNav.${key}Desc`)}</p>
            </div>
          </button>
        ))}
      </div>
    </section>
  )
}
