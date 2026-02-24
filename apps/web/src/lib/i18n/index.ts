import { writable, derived, get } from 'svelte/store';
import en from './en';
import uz from './uz';
import ru from './ru';
import kk from './kk';

export type Locale = 'en' | 'uz' | 'ru' | 'kk';
export type Messages = typeof en;

const dictionaries: Record<Locale, Messages> = { en, uz, ru, kk };

const STORAGE_KEY = 'zist_locale';

function detectLocale(): Locale {
  if (typeof localStorage === 'undefined') return 'ru'; // SSR default
  const stored = localStorage.getItem(STORAGE_KEY) as Locale | null;
  if (stored && stored in dictionaries) return stored;
  const browser = navigator.language.split('-')[0] as Locale;
  if (browser in dictionaries) return browser;
  return 'ru';
}

export const locale = writable<Locale>(
  typeof localStorage !== 'undefined' ? detectLocale() : 'ru'
);

// Persist locale changes.
if (typeof localStorage !== 'undefined') {
  locale.subscribe((l) => localStorage.setItem(STORAGE_KEY, l));
}

export const t = derived(locale, ($locale) => dictionaries[$locale]);

/** Switch the active locale. */
export function setLocale(l: Locale) {
  locale.set(l);
}

/** Access translations outside Svelte components. */
export function getT(): Messages {
  return dictionaries[get(locale)];
}
