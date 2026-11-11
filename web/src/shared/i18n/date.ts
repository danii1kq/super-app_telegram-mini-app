import type { Lang } from './index';

/**
 * Date and time formatting per interface language.
 *
 * A hardcoded 'ru-RU' leaks Russian month names into an English or Kazakh
 * screen, so every formatter takes the current language.
 */

const LOCALE: Record<Lang, string> = {
  ru: 'ru-RU',
  kk: 'kk-KZ',
  en: 'en-GB',
};

export function localeOf(lang: Lang): string {
  return LOCALE[lang];
}

export function formatDayMonth(value: string | Date, lang: Lang): string {
  return new Date(value).toLocaleDateString(localeOf(lang), { day: 'numeric', month: 'short' });
}

export function formatDateTime(value: string | Date, lang: Lang): string {
  return new Date(value).toLocaleString(localeOf(lang), {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatWeekday(value: string | Date, lang: Lang): string {
  return new Date(value).toLocaleDateString(localeOf(lang), {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

export function formatMonthDay(value: string | Date, lang: Lang): string {
  return new Date(value).toLocaleDateString(localeOf(lang), { day: 'numeric', month: 'long' });
}

export function formatTime(value: string | Date, lang: Lang): string {
  return new Date(value).toLocaleTimeString(localeOf(lang), { hour: '2-digit', minute: '2-digit' });
}

export function formatAgenda(value: string | Date, lang: Lang): string {
  return new Date(value).toLocaleString(localeOf(lang), {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}
