import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { deviceStorage } from '@/shared/telegram/sdk';

/**
 * Shell localisation: ru (default), kk, en. App content is localised by each
 * app once its own API supports it; the shell switches its own strings.
 */

export type Lang = 'ru' | 'kk' | 'en';

const LANG_KEY = 'srunal.lang';

const dict = {
  ru: {
    feed: 'Лента',
    chats: 'Чаты',
    apps: 'Приложения',
    profile: 'Профиль',
    back: 'Назад',
    retry: 'Повторить',
    loading: 'Загрузка',
    empty: 'Пока пусто',
    error: 'Не удалось загрузить',
    send: 'Отправить',
    save: 'Сохранить',
    cancel: 'Отмена',
    delete: 'Удалить',
    open: 'Открыть',
    soon: 'Скоро',
    planned: 'Запланировано',
  },
  kk: {
    feed: 'Лента',
    chats: 'Чаттар',
    apps: 'Қолданбалар',
    profile: 'Профиль',
    back: 'Артқа',
    retry: 'Қайталау',
    loading: 'Жүктелуде',
    empty: 'Әзірге бос',
    error: 'Жүктеу мүмкін болмады',
    send: 'Жіберу',
    save: 'Сақтау',
    cancel: 'Болдырмау',
    delete: 'Жою',
    open: 'Ашу',
    soon: 'Жақында',
    planned: 'Жоспарланған',
  },
  en: {
    feed: 'Feed',
    chats: 'Chats',
    apps: 'Apps',
    profile: 'Profile',
    back: 'Back',
    retry: 'Retry',
    loading: 'Loading',
    empty: 'Nothing here yet',
    error: 'Could not load',
    send: 'Send',
    save: 'Save',
    cancel: 'Cancel',
    delete: 'Delete',
    open: 'Open',
    soon: 'Soon',
    planned: 'Planned',
  },
} as const;

export type DictKey = keyof (typeof dict)['ru'];

interface I18nValue {
  lang: Lang;
  setLang: (next: Lang) => void;
  t: (key: DictKey) => string;
}

const I18nContext = createContext<I18nValue>({
  lang: 'ru',
  setLang: () => {},
  t: (key) => dict.ru[key],
});

export function useI18n(): I18nValue {
  return useContext(I18nContext);
}

export function useT(): (key: DictKey) => string {
  return useContext(I18nContext).t;
}

function loadLang(): Lang {
  const raw = deviceStorage.get(LANG_KEY);
  return raw === 'kk' || raw === 'en' || raw === 'ru' ? raw : 'ru';
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(loadLang);

  const setLang = useCallback((next: Lang) => {
    setLangState(next);
    deviceStorage.set(LANG_KEY, next);
    document.documentElement.lang = next;
  }, []);

  const t = useCallback((key: DictKey) => dict[lang][key], [lang]);

  const value = useMemo(() => ({ lang, setLang, t }), [lang, setLang, t]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}
