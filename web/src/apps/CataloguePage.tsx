import { useNavigate } from 'react-router-dom';
import { APPS } from '@/app/registry';
import { AppHeader, Icon, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';

/**
 * Catalogue: four apps as cards with an icon and a name.
 *
 * A grid of two columns, the way a launcher looks on a phone. Each card is the
 * whole tap target, so the name is never the only thing that opens the app.
 */
export function CataloguePage() {
  const navigate = useNavigate();
  const t = useT();

  return (
    <Screen header={<AppHeader title={t('apps')} leadingIcon="apps" />}>
      <div className="grid grid-cols-2 gap-3 pt-2">
        {APPS.map((app) => (
          <button
            key={app.id}
            type="button"
            onClick={() => navigate(app.path)}
            className="flex flex-col items-start gap-2 rounded-[18px] bg-surface p-4 text-left transition active:scale-[0.98]"
          >
            <span className="flex size-12 items-center justify-center rounded-2xl bg-accent-soft text-accent">
              <Icon name={app.icon} size={26} />
            </span>
            <span className="text-body-strong text-ink">{app.title}</span>
            <span className="text-caption text-ink-2">{t(app.tagline)}</span>
          </button>
        ))}
      </div>

      <div className="flex items-start gap-2 px-1 pt-4 text-caption text-ink-2">
        <Icon name="info" size={16} className="pt-0.5" />
        <span>{t('appsHint')}</span>
      </div>
    </Screen>
  );
}
