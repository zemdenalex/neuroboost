import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthContext } from '../../contexts/AuthContext'
import { useMediaQuery } from '../../hooks/useMediaQuery'
import { showToast } from '../../components/ui/Toast'
import type { UserSettings } from '../../api/auth'
import { createDebouncedSaver, type DebouncedSaver } from '../../lib/autoSave/debouncedSaver'
import { SessionSection } from './sections/SessionSection'
import { HintStyleSection } from './sections/HintStyleSection'
import { LayoutStyleSection } from './sections/LayoutStyleSection'
import { MobileNavSection } from './sections/MobileNavSection'
import { UIScaleSection } from './sections/UIScaleSection'
import { WorkHoursSection } from './sections/WorkHoursSection'
import { FeatureTogglesSection } from './sections/FeatureTogglesSection'
import { RecurringScopeSection } from './sections/RecurringScopeSection'
import { DataSection } from './sections/DataSection'
import { RemindersSection } from './sections/RemindersSection'
import { QuickTaskSection } from './sections/QuickTaskSection'
import { RegionalSection } from './sections/RegionalSection'
import { CalendarsSection } from '../../components/Calendars/CalendarsSection'

/** The subset of the profile this page edits. */
type ProfilePatch = { display_name?: string; timezone?: string; locale?: string }

/**
 * Settings page.
 *
 * 873 lines and thirteen inline sections on 2026-08-14; now a list of sections
 * plus the one thing they genuinely share — the debounced saver, whose
 * flush-on-unmount is what stops a change being lost when the user navigates
 * away inside the 300ms window.
 *
 * The error banner survives for auto-save failures only. Sections report their
 * own failures through showToast, following CalendarsSection.
 */
export default function Settings() {
  const { t } = useTranslation('settings')
  const { updateSettings, updateProfile } = useAuthContext()
  const isMobile = useMediaQuery('(max-width: 767px)')
  const [error, setError] = useState<string | null>(null)

  // Debounced auto-save. The machinery lives in lib/autoSave so it can be
  // tested: both copies that used to sit here cancelled their timer on unmount
  // under a comment claiming they flushed it, so a change followed by leaving
  // the page inside 300ms was lost silently. See debouncedSaver.test.ts.
  //
  // Held in refs and built once: rebuilding the saver on re-render would drop
  // the pending patch it was holding.
  const settingsSaverRef = useRef<DebouncedSaver<UserSettings> | null>(null)
  const profileSaverRef = useRef<DebouncedSaver<ProfilePatch> | null>(null)

  // Callbacks change identity every render; the savers are created once, so
  // they read through this ref rather than closing over the first render's
  // versions.
  const handlersRef = useRef({ updateSettings, updateProfile, showToast, setError, t })
  handlersRef.current = { updateSettings, updateProfile, showToast, setError, t }

  if (settingsSaverRef.current === null) {
    settingsSaverRef.current = createDebouncedSaver<UserSettings>({
      save: (patch) => handlersRef.current.updateSettings(patch),
      onSaved: () => handlersRef.current.showToast(handlersRef.current.t('saved')),
      onError: (err) => {
        console.error('Auto-save failed:', err)
        handlersRef.current.setError(handlersRef.current.t('error.saveSettings'))
      },
      onFlushError: (err) => console.error('Auto-save flush failed:', err),
    })
  }

  if (profileSaverRef.current === null) {
    profileSaverRef.current = createDebouncedSaver<ProfilePatch>({
      save: (patch) => handlersRef.current.updateProfile(patch),
      onSaved: () => handlersRef.current.showToast(handlersRef.current.t('saved')),
      onError: (err) => {
        console.error('Auto-save failed:', err)
        handlersRef.current.setError(handlersRef.current.t('error.saveSettings'))
      },
      onFlushError: (err) => console.error('Auto-save flush failed:', err),
    })
  }

  const autoSaveSettings = useCallback((patch: Partial<UserSettings>) => {
    settingsSaverRef.current?.schedule(patch)
  }, [])

  const autoSaveProfile = useCallback((patch: Partial<ProfilePatch>) => {
    profileSaverRef.current?.schedule(patch)
  }, [])

  // 🔴 Flush, not cancel. This is the whole point of the extraction: what the
  // old comment here promised and the old body did not do.
  useEffect(() => {
    return () => {
      settingsSaverRef.current?.flush()
      profileSaverRef.current?.flush()
    }
  }, [])

  // Settings state - initialize from user settings or defaults

  // Feature toggles


  // Apply header style immediately when changed

  // Hint style is localStorage-only (v1); broadcast so the live OnboardingProvider updates immediately.




  return (
    <div className="max-w-3xl mx-auto p-6 space-y-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-mono font-bold text-white">{t('title')}</h1>
        </div>

        {error && (
          <div className="p-3 bg-red-900/30 border border-red-800 rounded-lg text-red-400 text-sm">
            {error}
          </div>
        )}

        <LayoutStyleSection />

        <HintStyleSection />

        {/* The isMobile gate stays here: it decides whether the section exists. */}
        {isMobile && <MobileNavSection />}

        <UIScaleSection autoSave={autoSaveSettings} />

        <RecurringScopeSection autoSave={autoSaveSettings} />

        <QuickTaskSection autoSave={autoSaveSettings} />

        <RemindersSection autoSave={autoSaveSettings} />

        <CalendarsSection />

        <WorkHoursSection autoSave={autoSaveSettings} />

        <RegionalSection autoSaveProfile={autoSaveProfile} />

        <FeatureTogglesSection autoSave={autoSaveSettings} />

        <DataSection />

        <SessionSection />
    </div>
  )
}