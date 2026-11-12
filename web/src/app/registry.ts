/**
 * App registry: the catalogue of the five apps plus the SruNal core sections.
 *
 * Each app owns its route and its lazy-loaded module. Adding an app means adding
 * one entry here and its folder under src/apps/<id>.
 *
 * Titles are brand names and stay as they are. Taglines are copy, so they are
 * dictionary keys: an English interface must not show a Russian sentence.
 */

import type { DictKey } from '@/shared/i18n';

export type AppId = 'doc' | 'day' | 'pulse' | 'notes' | 'split' | 'farm';

export interface AppMeta {
  id: AppId;
  /** Route inside the SruNal shell. */
  path: string;
  title: string;
  /** One sentence shown on the catalogue card. */
  tagline: DictKey;
  icon: string;
  /** False while the app is not implemented: the card says "planned". */
  ready: boolean;
}

export const APPS: readonly AppMeta[] = [
  {
    id: 'farm',
    path: '/apps/farm',
    title: 'SruFarm',
    tagline: 'appFarmTagline',
    icon: 'potted_plant',
    ready: true,
  },
  {
    id: 'doc',
    path: '/apps/doc',
    title: 'SruDoc',
    tagline: 'appDocTagline',
    icon: 'description',
    ready: true,
  },
  {
    id: 'day',
    path: '/apps/day',
    title: 'SruDay',
    tagline: 'appDayTagline',
    icon: 'calendar_month',
    ready: true,
  },
  {
    id: 'pulse',
    path: '/apps/pulse',
    title: 'SruPulse',
    tagline: 'appPulseTagline',
    icon: 'bolt',
    ready: true,
  },
  {
    id: 'notes',
    path: '/apps/notes',
    title: 'SruNotes',
    tagline: 'appNotesTagline',
    icon: 'sticky_note_2',
    ready: true,
  },
  {
    id: 'split',
    path: '/apps/split',
    title: 'SruSplit',
    tagline: 'appSplitTagline',
    icon: 'receipt_long',
    ready: true,
  },
] as const;

export function appById(id: AppId): AppMeta {
  const found = APPS.find((app) => app.id === id);
  if (!found) throw new Error(`unknown app: ${id}`);
  return found;
}
