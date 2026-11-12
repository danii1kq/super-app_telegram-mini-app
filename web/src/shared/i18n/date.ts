import type { Lang } from './index';

/**
 * Date and time formatting per interface language.
 *
 * A hardcoded 'ru-RU' leaks Russian month names into an English or Kazakh
 * screen, so every formatter takes the current language.
 *
 * Kazakh months are spelled out: the browser's ICU falls back to Russian month
 * names for kk-KZ in some builds, which showed "12 нояб." on a Kazakh screen.
 */

const LOCALE: Record<Lang, string> = {
  ru: 'ru-RU',
  kk: 'kk-KZ',
  en: 'en-GB',
};

const KK_MONTHS_SHORT = [
  'қаң', 'ақп', 'нау', 'сәу', 'мам', 'мау', 'шіл', 'там', 'қыр', 'қаз', 'қар', 'жел',
];

const KK_MONTHS_LONG = [
  'қаңтар', 'ақпан', 'наурыз', 'сәуір', 'мамыр', 'маусым',
  'шілде', 'тамыз', 'қыркүйек', 'қазан', 'қараша', 'желтоқсан',
];

const KK_WEEKDAYS_SHORT = ['жс', 'дс', 'сс', 'ср', 'бс', 'жм', 'сн'];

export function localeOf(lang: Lang): string {
  return LOCALE[lang];
}

function parts(value: string | Date): { date: Date; day: number; month: number } {
  const date = new Date(value);
  return { date, day: date.getDate(), month: date.getMonth() };
}

export function formatDayMonth(value: string | Date, lang: Lang): string {
  if (lang === 'kk') {
    const { day, month } = parts(value);
    return `${day} ${KK_MONTHS_SHORT[month]}`;
  }
  return new Date(value).toLocaleDateString(localeOf(lang), { day: 'numeric', month: 'short' });
}

export function formatDateTime(value: string | Date, lang: Lang): string {
  if (lang === 'kk') {
    const { date, day, month } = parts(value);
    const time = date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    return `${day} ${KK_MONTHS_SHORT[month]}, ${time}`;
  }
  return new Date(value).toLocaleString(localeOf(lang), {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatWeekday(value: string | Date, lang: Lang): string {
  if (lang === 'kk') {
    const { date, day, month } = parts(value);
    // getDay: 0 is Sunday, the Kazakh list starts on Sunday too.
    return `${KK_WEEKDAYS_SHORT[date.getDay()]} ${day} ${KK_MONTHS_SHORT[month]}`;
  }
  return new Date(value).toLocaleDateString(localeOf(lang), {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

export function formatMonthDay(value: string | Date, lang: Lang): string {
  if (lang === 'kk') {
    const { day, month } = parts(value);
    return `${day} ${KK_MONTHS_LONG[month]}`;
  }
  return new Date(value).toLocaleDateString(localeOf(lang), { day: 'numeric', month: 'long' });
}

export function formatTime(value: string | Date, lang: Lang): string {
  return new Date(value).toLocaleTimeString(localeOf(lang), { hour: '2-digit', minute: '2-digit' });
}

export function formatAgenda(value: string | Date, lang: Lang): string {
  if (lang === 'kk') {
    const { date, day, month } = parts(value);
    const time = date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    return `${KK_WEEKDAYS_SHORT[date.getDay()]}, ${day} ${KK_MONTHS_SHORT[month]}, ${time}`;
  }
  return new Date(value).toLocaleString(localeOf(lang), {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/** A month name with the year, for the report header. */
export function formatMonthYear(key: string, lang: Lang): string {
  const [year = 2026, month = 1] = key.split('-').map(Number);
  if (lang === 'kk') {
    return `${KK_MONTHS_LONG[month - 1]} ${year}`;
  }
  return new Date(year, month - 1, 1).toLocaleDateString(localeOf(lang), {
    month: 'long',
    year: 'numeric',
  });
}
