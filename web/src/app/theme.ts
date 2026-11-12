import { deviceStorage, tg, versionAtLeast } from '@/shared/telegram/sdk';

/**
 * Interface theme: follow Telegram (default) or force light/dark.
 *
 * "auto"  - the palette comes from the Telegram theme variables;
 * "light" - fixed light palette even if Telegram is dark;
 * "dark"  - fixed dark palette.
 *
 * The choice is stored on the device and applied through data-scheme on <html>;
 * the CSS resolves the semantic tokens per scheme. applyThemeMode is called both
 * at start and on every change, so the switch takes effect immediately.
 */

export type ThemeMode = 'auto' | 'light' | 'dark';

const THEME_KEY = 'srunal.theme';

export function loadThemeMode(): ThemeMode {
  const raw = deviceStorage.get(THEME_KEY);
  return raw === 'light' || raw === 'dark' || raw === 'auto' ? raw : 'auto';
}

export function saveThemeMode(mode: ThemeMode): void {
  deviceStorage.set(THEME_KEY, mode);
}

/** Resolves the mode to a concrete scheme ("auto" follows Telegram). */
export function resolveScheme(mode: ThemeMode = loadThemeMode()): 'light' | 'dark' {
  if (mode === 'light' || mode === 'dark') return mode;
  if (tg?.colorScheme === 'dark') return 'dark';
  if (tg?.colorScheme === 'light') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/** Telegram's own dark palette, used when Telegram is not driving the theme. */
const TG_DARK = {
  bg: '#0e1621',
  surface: '#17212b',
  surface2: '#17212b',
  ink: '#ffffff',
  ink2: '#8a9bb0',
  hint: '#8a9bb0',
  link: '#6ab3f3',
  button: '#5288c1',
  line: '#2a3743',
};

const TG_LIGHT = {
  bg: '#efeff4',
  surface: '#ffffff',
  surface2: '#ffffff',
  ink: '#000000',
  ink2: '#8e8e93',
  hint: '#8e8e93',
  link: '#2481cc',
  button: '#2481cc',
  line: 'rgba(60, 60, 67, 0.2)',
};

/**
 * Writes the palette onto <html> as inline custom properties.
 *
 * Inside Telegram with "auto" the --tg-theme-* variables already carry the
 * client palette and the CSS falls back to them, so nothing is written. A forced
 * light/dark choice has to override them, which is what this does.
 */
function applyPalette(mode: ThemeMode, scheme: 'light' | 'dark'): void {
  const root = document.documentElement;
  const followTelegram = mode === 'auto' && Boolean(tg?.themeParams);

  if (followTelegram) {
    for (const name of Object.keys(OVERRIDES)) root.style.removeProperty(name);
    return;
  }

  const palette = scheme === 'dark' ? TG_DARK : TG_LIGHT;
  root.style.setProperty('--app-bg', palette.bg);
  root.style.setProperty('--app-surface', palette.surface);
  root.style.setProperty('--app-surface-2', palette.surface2);
  root.style.setProperty('--app-ink', palette.ink);
  root.style.setProperty('--app-ink-2', palette.ink2);
  root.style.setProperty('--app-hint', palette.hint);
  root.style.setProperty('--app-link', palette.link);
  root.style.setProperty('--app-btn', palette.button);
  root.style.setProperty('--app-line', palette.line);
}

const OVERRIDES: Record<string, string> = {
  '--app-bg': '',
  '--app-surface': '',
  '--app-surface-2': '',
  '--app-ink': '',
  '--app-ink-2': '',
  '--app-hint': '',
  '--app-link': '',
  '--app-btn': '',
  '--app-line': '',
};

export function applyThemeMode(mode: ThemeMode = loadThemeMode()): void {
  const scheme = resolveScheme(mode);
  document.documentElement.dataset.scheme = scheme;
  document.documentElement.dataset.themeMode = mode;
  applyPalette(mode, scheme);

  // The browser chrome follows the scheme even outside Telegram.
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) meta.setAttribute('content', scheme === 'dark' ? '#17212b' : '#efeff4');

  if (!tg) return;

  const followTelegram = mode === 'auto' && versionAtLeast('7.0');
  const bar = followTelegram ? 'secondary_bg_color' : scheme === 'dark' ? '#17212b' : '#efeff4';
  try {
    if (versionAtLeast('6.1')) {
      tg.setHeaderColor(bar);
      tg.setBackgroundColor(bar);
    }
    if (versionAtLeast('7.10')) {
      tg.setBottomBarColor?.(bar);
    }
  } catch {
    /* older clients keep their defaults */
  }
}

/** Applies the stored theme and keeps "auto" in sync with Telegram and the OS. */
export function initTheme(): () => void {
  applyThemeMode();

  const media = window.matchMedia('(prefers-color-scheme: dark)');
  const onSystemChange = () => {
    if (loadThemeMode() === 'auto') applyThemeMode();
  };
  media.addEventListener('change', onSystemChange);

  const client = tg;
  if (!client) return () => media.removeEventListener('change', onSystemChange);

  const onThemeChanged = () => applyThemeMode();
  client.onEvent('themeChanged', onThemeChanged);
  return () => {
    media.removeEventListener('change', onSystemChange);
    client.offEvent('themeChanged', onThemeChanged);
  };
}
