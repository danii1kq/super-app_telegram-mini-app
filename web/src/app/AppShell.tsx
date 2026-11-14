import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { Icon } from '@/shared/ui';
import { useT, type DictKey } from '@/shared/i18n';

const TABS: readonly { to: string; label: DictKey; icon: string }[] = [
  { to: '/feed', label: 'feed', icon: 'dynamic_feed' },
  { to: '/calendar', label: 'calendar', icon: 'calendar_month' },
  { to: '/apps', label: 'apps', icon: 'apps' },
  { to: '/profile', label: 'profile', icon: 'person' },
];

/**
 * Shell around every section. The tab bar is hidden inside an app space and in
 * a single conversation: both own the screen, and the tab bar would cover the
 * message composer.
 */
export function AppShell() {
  const t = useT();
  const location = useLocation();
  const insideApp = location.pathname.startsWith('/apps/');
  const insideChat = /^\/chats\/[^/]+/.test(location.pathname);
  const hideTabs = insideApp || insideChat;

  return (
    <div className="flex min-h-screen flex-col bg-page">
      <Outlet />
      {hideTabs ? null : (
        <nav
          className="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-surface/95 backdrop-blur-xl"
          style={{ paddingBottom: 'var(--safe-bottom)' }}
        >
          <div className="mx-auto flex w-full max-w-[480px] items-stretch">
            {TABS.map((tab) => (
              <NavLink
                key={tab.to}
                to={tab.to}
                className={({ isActive }) =>
                  `flex flex-1 flex-col items-center gap-0.5 py-2 ${
                    isActive ? 'text-accent' : 'text-ink-2'
                  }`
                }
              >
                <Icon name={tab.icon} size={24} />
                <span className="text-[11px] font-medium">{t(tab.label)}</span>
              </NavLink>
            ))}
          </div>
        </nav>
      )}
    </div>
  );
}
