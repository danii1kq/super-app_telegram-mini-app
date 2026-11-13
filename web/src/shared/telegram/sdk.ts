/**
 * Thin typed wrapper around the Telegram WebApp SDK.
 *
 * Every call is guarded: outside Telegram (browser preview) the helpers degrade
 * to safe no-ops and the app shows the gate screen.
 */

export const tg: TelegramWebApp | undefined = window.Telegram?.WebApp;

/** True when native Telegram buttons (MainButton/BackButton) are usable. */
export function hasNativeButtons(): boolean {
  return Boolean(tg && tg.initData && tg.platform !== 'unknown');
}

/** Raw initData string; empty outside Telegram. */
export function initData(): string {
  return tg?.initData ?? '';
}

/** True when the app runs inside Telegram with a signed payload. */
export function isMiniApp(): boolean {
  return Boolean(tg?.initData);
}

export function telegramUser(): TelegramUser | undefined {
  const user = tg?.initDataUnsafe?.user;
  return user && user.id ? user : undefined;
}

/** The `startapp` parameter of a Mini App deep link. */
export function startParam(): string {
  return tg?.initDataUnsafe?.start_param ?? '';
}

export function requestBotWriteAccess(): Promise<boolean> {
  return new Promise(resolve=>{if(!tg?.requestWriteAccess){resolve(false);return;}try{tg.requestWriteAccess(resolve);}catch{resolve(false);}});
}

export function versionAtLeast(version: string): boolean {
  try {
    return tg?.isVersionAtLeast?.(version) ?? false;
  } catch {
    return false;
  }
}

/** Applies safe areas and subscribes to their changes. */
export function bootstrapTelegram(): () => void {
  setSafeArea();
  if (!tg) return () => {};

  tg.onEvent('safeAreaChanged', setSafeArea);
  tg.onEvent('contentSafeAreaChanged', setSafeArea);

  return () => {
    tg.offEvent('safeAreaChanged', setSafeArea);
    tg.offEvent('contentSafeAreaChanged', setSafeArea);
  };
}

/** Device notch + Telegram UI insets, with hard fallbacks. */
export function setSafeArea(): void {
  const top = Math.max(tg?.contentSafeAreaInset?.top ?? 0, tg?.safeAreaInset?.top ?? 0);
  const bottom = Math.max(tg?.contentSafeAreaInset?.bottom ?? 0, tg?.safeAreaInset?.bottom ?? 0);
  const root = document.documentElement;
  root.style.setProperty('--safe-top', `${top}px`);
  root.style.setProperty('--safe-bottom', `${bottom}px`);
}

export const haptic = {
  tap(): void {
    if (!versionAtLeast('6.1')) return;
    tg?.HapticFeedback?.impactOccurred('light');
  },
  select(): void {
    if (!versionAtLeast('6.1')) return;
    tg?.HapticFeedback?.selectionChanged();
  },
  impact(style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft' = 'medium'): void {
    if (!versionAtLeast('6.1')) return;
    tg?.HapticFeedback?.impactOccurred(style);
  },
  success(): void {
    if (!versionAtLeast('6.1')) return;
    tg?.HapticFeedback?.notificationOccurred('success');
  },
  error(): void {
    if (!versionAtLeast('6.1')) return;
    tg?.HapticFeedback?.notificationOccurred('error');
  },
  warning(): void {
    if (!versionAtLeast('6.1')) return;
    tg?.HapticFeedback?.notificationOccurred('warning');
  },
};

export function showAlert(message: string): void {
  if (tg && versionAtLeast('6.2')) {
    tg.showAlert(message);
    return;
  }
  window.alert(message);
}

export function showConfirm(message: string): Promise<boolean> {
  if (tg && versionAtLeast('6.2')) {
    return new Promise((resolve) => tg.showConfirm(message, (confirmed) => resolve(confirmed)));
  }
  return Promise.resolve(window.confirm(message));
}

export function openLink(url: string): void {
  if (tg) {
    tg.openLink(url);
    return;
  }
  window.open(url, '_blank', 'noopener');
}

export function openTelegramLink(url: string): void {
  if (tg && isMiniApp()) {
    tg.openTelegramLink(url);
    return;
  }
  window.open(url, '_blank', 'noopener');
}

export function copyText(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(text);
  }
  const area = document.createElement('textarea');
  area.value = text;
  area.style.position = 'fixed';
  area.style.opacity = '0';
  document.body.appendChild(area);
  area.select();
  try {
    document.execCommand('copy');
  } finally {
    document.body.removeChild(area);
  }
  return Promise.resolve();
}

/** Warns the user before closing the app while a draft is unsaved. */
export function setClosingConfirmation(enabled: boolean): void {
  try {
    if (enabled) {
      tg?.enableClosingConfirmation();
    } else {
      tg?.disableClosingConfirmation();
    }
  } catch {
    /* older clients */
  }
}

/** DeviceStorage ships with Bot API 9.0; older clients get localStorage. */
function deviceStorageAvailable(): boolean {
  return Boolean(tg?.DeviceStorage) && versionAtLeast('9.0');
}

export const deviceStorage = {
  get(key: string): string | null {
    try {
      const local = window.localStorage.getItem(key);
      if (local !== null && local !== '') return local;
    } catch {
      /* ignore */
    }
    if (deviceStorageAvailable()) {
      try {
        const value = tg?.DeviceStorage?.getItem(key);
        if (value !== undefined && value !== null && value !== '') return value;
      } catch {
        /* ignore */
      }
    }
    return null;
  },
  set(key: string, value: string): void {
    try {
      window.localStorage.setItem(key, value);
    } catch {
      /* ignore */
    }
    if (deviceStorageAvailable()) {
      try {
        tg?.DeviceStorage?.setItem(key, value);
      } catch {
        /* ignore */
      }
    }
  },
  remove(key: string): void {
    try {
      window.localStorage.removeItem(key);
    } catch {
      /* ignore */
    }
    if (deviceStorageAvailable()) {
      try {
        tg?.DeviceStorage?.removeItem(key);
      } catch {
        /* ignore */
      }
    }
  },
};
