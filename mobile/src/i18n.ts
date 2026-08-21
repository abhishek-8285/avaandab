import AsyncStorage from '@react-native-async-storage/async-storage';

import en from './locales/en.json';
import hi from './locales/hi.json';
import ta from './locales/ta.json';
import te from './locales/te.json';
import kn from './locales/kn.json';
import mr from './locales/mr.json';
import gu from './locales/gu.json';

// Lightweight i18n implementation with react-i18next compatible API
// Tries to use i18next if available, else falls back to simple lookup
export type SupportedLocale = 'en' | 'hi' | 'ta' | 'te' | 'kn' | 'mr' | 'gu';

const resources: Record<string, any> = {
  en: { translation: en },
  hi: { translation: hi },
  ta: { translation: ta },
  te: { translation: te },
  kn: { translation: kn },
  mr: { translation: mr },
  gu: { translation: gu },
};

const LANGUAGE_KEY = '@avandab_language';

let currentLocale: SupportedLocale = 'en';
let i18nInstance: any = null;

// Try to init i18next if installed
try {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const i18next = require('i18next');
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { initReactI18next } = require('react-i18next');
  i18nInstance = i18next;
  i18next.use(initReactI18next).init({
    resources,
    lng: 'en',
    fallbackLng: 'en',
    compatibilityJSON: 'v3',
    interpolation: { escapeValue: false },
  });
  // Load persisted language
  AsyncStorage.getItem(LANGUAGE_KEY)
    .then((saved) => {
      if (saved && resources[saved]) {
        i18next.changeLanguage(saved);
        currentLocale = saved as SupportedLocale;
      }
    })
    .catch(() => {});
} catch {
  // i18next not installed — use lightweight fallback
  AsyncStorage.getItem(LANGUAGE_KEY)
    .then((saved) => {
      if (saved && resources[saved]) {
        currentLocale = saved as SupportedLocale;
      }
    })
    .catch(() => {});
}

export function t(key: string, fallback?: string): string {
  if (i18nInstance) {
    const val = i18nInstance.t(key);
    // i18next returns key if missing; fallback to en
    if (val === key) {
      return (en as any)[key] || fallback || key;
    }
    return val;
  }
  const dict = resources[currentLocale]?.translation || en;
  return dict[key] ?? (en as any)[key] ?? fallback ?? key;
}

export async function setLocale(locale: SupportedLocale): Promise<void> {
  if (!resources[locale]) return;
  currentLocale = locale;
  await AsyncStorage.setItem(LANGUAGE_KEY, locale);
  if (i18nInstance) {
    await i18nInstance.changeLanguage(locale);
  }
}

export function getLocale(): SupportedLocale {
  if (i18nInstance && i18nInstance.language) {
    return i18nInstance.language as SupportedLocale;
  }
  return currentLocale;
}

export function getSupportedLocales(): SupportedLocale[] {
  return Object.keys(resources) as SupportedLocale[];
}

const i18n = i18nInstance || {
  t,
  language: currentLocale,
  changeLanguage: async (lng: string) => setLocale(lng as SupportedLocale),
  use: () => i18n,
  init: () => Promise.resolve(),
};

export default i18n;
