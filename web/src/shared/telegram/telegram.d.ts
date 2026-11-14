/** Minimal typings for the Telegram WebApp SDK surface this app uses. */

interface TelegramUser {
  id: number;
  first_name?: string;
  last_name?: string;
  username?: string;
  language_code?: string;
  photo_url?: string;
}

interface TelegramSafeAreaInset {
  top?: number;
  bottom?: number;
  left?: number;
  right?: number;
}

interface TelegramMainButton {
  setText(text: string): void;
  onClick(listener: () => void): void;
  offClick(listener: () => void): void;
  show(): void;
  hide(): void;
  enable(): void;
  disable(): void;
  showProgress(leaveActive?: boolean): void;
  hideProgress(): void;
}

interface TelegramBackButton {
  onClick(listener: () => void): void;
  offClick(listener: () => void): void;
  show(): void;
  hide(): void;
}

interface TelegramWebApp {
  initData: string;
  initDataUnsafe?: {
    user?: TelegramUser;
    start_param?: string;
  };
  platform?: string;
  colorScheme?: 'light' | 'dark';
  themeParams?: Record<string, string>;
  safeAreaInset?: TelegramSafeAreaInset;
  contentSafeAreaInset?: TelegramSafeAreaInset;
  MainButton?: TelegramMainButton;
  BackButton?: TelegramBackButton;
  HapticFeedback?: {
    impactOccurred(style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft'): void;
    notificationOccurred(type: 'error' | 'success' | 'warning'): void;
    selectionChanged(): void;
  };
  DeviceStorage?: {
    getItem(key: string): string | null | undefined;
    setItem(key: string, value: string): void;
    removeItem(key: string): void;
  };
  isVersionAtLeast?(version: string): boolean;
  requestWriteAccess?(callback: (allowed: boolean) => void): void;
  openInvoice?(url: string, callback?: (status: 'paid' | 'cancelled' | 'failed' | 'pending') => void): void;
  onEvent(event: string, handler: () => void): void;
  offEvent(event: string, handler: () => void): void;
  setHeaderColor(color: string): void;
  setBackgroundColor(color: string): void;
  setBottomBarColor?(color: string): void;
  showAlert(message: string): void;
  showConfirm(message: string, callback: (confirmed: boolean) => void): void;
  openLink(url: string): void;
  openTelegramLink(url: string): void;
  enableClosingConfirmation(): void;
  disableClosingConfirmation(): void;
  ready(): void;
  expand(): void;
}

interface Window {
  Telegram?: { WebApp?: TelegramWebApp };
}
