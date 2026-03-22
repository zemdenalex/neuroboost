import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

import enCommon from './locales/en/common.json'
import enAuth from './locales/en/auth.json'
import enCalendar from './locales/en/calendar.json'
import enTasks from './locales/en/tasks.json'
import enSettings from './locales/en/settings.json'
import enProfile from './locales/en/profile.json'
import enFeedback from './locales/en/feedback.json'
import enHome from './locales/en/home.json'

import ruCommon from './locales/ru/common.json'
import ruAuth from './locales/ru/auth.json'
import ruCalendar from './locales/ru/calendar.json'
import ruTasks from './locales/ru/tasks.json'
import ruSettings from './locales/ru/settings.json'
import ruProfile from './locales/ru/profile.json'
import ruFeedback from './locales/ru/feedback.json'
import ruHome from './locales/ru/home.json'

const ns = ['common', 'auth', 'calendar', 'tasks', 'settings', 'profile', 'feedback', 'home'] as const

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        common: enCommon,
        auth: enAuth,
        calendar: enCalendar,
        tasks: enTasks,
        settings: enSettings,
        profile: enProfile,
        feedback: enFeedback,
        home: enHome,
      },
      ru: {
        common: ruCommon,
        auth: ruAuth,
        calendar: ruCalendar,
        tasks: ruTasks,
        settings: ruSettings,
        profile: ruProfile,
        feedback: ruFeedback,
        home: ruHome,
      },
    },
    fallbackLng: 'en',
    defaultNS: 'common',
    ns: [...ns],
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'neuroboost-locale',
      caches: ['localStorage'],
    },
  })

export default i18n
