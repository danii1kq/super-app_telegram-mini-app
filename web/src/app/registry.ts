/**
 * App registry: the catalogue of the five apps plus the SruNal core sections.
 *
 * Each app owns its route and its lazy-loaded module. Adding an app means adding
 * one entry here and its folder under src/apps/<id>.
 */

export type AppId = 'doc' | 'day' | 'pulse' | 'notes' | 'split';

export interface AppMeta {
  id: AppId;
  /** Route inside the SruNal shell. */
  path: string;
  title: string;
  /** One sentence shown on the catalogue card. */
  tagline: string;
  icon: string;
  /** False while the app is not implemented: the card says "planned". */
  ready: boolean;
}

export const APPS: readonly AppMeta[] = [
  {
    id: 'doc',
    path: '/apps/doc',
    title: 'SruDoc',
    tagline: 'Документы и заказы: подготовка, оплата, результат',
    icon: 'description',
    ready: true,
  },
  {
    id: 'day',
    path: '/apps/day',
    title: 'SruDay',
    tagline: 'Личный и групповой календарь',
    icon: 'calendar_month',
    ready: true,
  },
  {
    id: 'pulse',
    path: '/apps/pulse',
    title: 'SruPulse',
    tagline: 'Свободен сейчас: собрать компанию за минуту',
    icon: 'bolt',
    ready: true,
  },
  {
    id: 'notes',
    path: '/apps/notes',
    title: 'SruNotes',
    tagline: 'Простые заметки, которыми можно поделиться',
    icon: 'sticky_note_2',
    ready: true,
  },
  {
    id: 'split',
    path: '/apps/split',
    title: 'SruSplit',
    tagline: 'Общие расходы и честный расчёт долей',
    icon: 'receipt_long',
    ready: true,
  },
] as const;

export function appById(id: AppId): AppMeta {
  const found = APPS.find((app) => app.id === id);
  if (!found) throw new Error(`unknown app: ${id}`);
  return found;
}
