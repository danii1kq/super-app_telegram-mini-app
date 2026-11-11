import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { api, ApiError, clearSessionToken, setSessionToken, setUnauthorizedHandler } from '@/shared/api/client';
import { initData, isMiniApp, telegramUser } from '@/shared/telegram/sdk';
import { useT } from '@/shared/i18n';

/**
 * Session: one Telegram-verified login per Mini App launch. The token lives in
 * memory only; a reload performs a fresh verified login.
 */

export interface Me {
  id: string;
  public_name: string;
  username?: string;
  photo_url?: string;
  lang: 'ru' | 'kk' | 'en';
  theme: 'auto' | 'light' | 'dark';
  timezone: string;
}

interface SessionValue {
  me: Me | null;
  status: 'loading' | 'ready' | 'error' | 'outside';
  error?: string;
  retry: () => void;
  logout: () => void;
}

const SessionContext = createContext<SessionValue>({
  me: null,
  status: 'loading',
  retry: () => {},
  logout: () => {},
});

export function useSession(): SessionValue {
  return useContext(SessionContext);
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const t = useT();
  const [me, setMe] = useState<Me | null>(null);
  const [status, setStatus] = useState<SessionValue['status']>('loading');
  const [error, setError] = useState<string | undefined>(undefined);
  const [attempt, setAttempt] = useState(0);

  const login = useCallback(async () => {
    const signed = initData();
    if (!signed) {
      // Outside Telegram (browser preview) the dev login keeps the app usable.
      // The server mounts it only with SRUNAL_DEV_INSECURE_KEYS=1.
      if (import.meta.env.DEV) {
        try {
          const response = await api<{ token: string; me: Me }>('/auth/dev', {
            method: 'POST',
            body: { name: 'Разработчик' },
          });
          setSessionToken(response.token);
          setMe(response.me);
          setStatus('ready');
          return;
        } catch {
          setStatus('outside');
          return;
        }
      }
      setStatus('outside');
      return;
    }
    setStatus('loading');
    setError(undefined);
    try {
      const response = await api<{ token: string; me: Me }>('/auth/telegram', {
        method: 'POST',
        body: { init_data: signed },
      });
      setSessionToken(response.token);
      setMe(response.me);
      setStatus('ready');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('loginFailed'));
      setStatus('error');
    }
  }, [t]);

  useEffect(() => {
    void login();
  }, [login, attempt]);

  useEffect(() => {
    setUnauthorizedHandler(() => {
      setMe(null);
      setStatus('error');
      setError(t('sessionExpired'));
    });
    return () => setUnauthorizedHandler(undefined);
  }, [t]);

  const retry = useCallback(() => setAttempt((value) => value + 1), []);

  const logout = useCallback(() => {
    clearSessionToken();
    setMe(null);
    setStatus('outside');
  }, []);

  const value = useMemo(
    () => ({ me, status, error, retry, logout }),
    [me, status, error, retry, logout],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

/** True when the app runs inside Telegram with a signed payload. */
export function useInsideTelegram(): boolean {
  return isMiniApp();
}

export function useTelegramName(): string {
  const user = telegramUser();
  return user?.first_name ?? '';
}
