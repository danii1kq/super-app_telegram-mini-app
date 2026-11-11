import { useNavigate } from 'react-router-dom';
import { APPS } from '@/app/registry';
import { AppHeader, Card, Chip, Hairline, Icon, Row, Screen, SectionLabel } from '@/shared/ui';
import { useT } from '@/shared/i18n';

/** Catalogue: the five apps of the super app. */
export function CataloguePage() {
  const navigate = useNavigate();
  const t = useT();

  return (
    <Screen header={<AppHeader title={t('apps')} leadingIcon="apps" />}>
      <SectionLabel>{t('apps')}</SectionLabel>
      <Card>
        {APPS.map((app, index) => (
          <div key={app.id}>
            {index > 0 ? <Hairline /> : null}
            <Row
              icon={app.icon}
              title={app.title}
              subtitle={app.tagline}
              onClick={app.ready ? () => navigate(app.path) : undefined}
              trailing={
                app.ready ? undefined : (
                  <Chip tone="neutral" icon="schedule">
                    {t('planned')}
                  </Chip>
                )
              }
              disabled={!app.ready}
            />
          </div>
        ))}
      </Card>

      <div className="flex items-start gap-2 px-1 pt-4 text-caption text-ink-2">
        <Icon name="info" size={16} className="pt-0.5" />
        <span>
          Каждое приложение открывается в своём пространстве. Связи между ними включаются отдельно и
          только с вашего подтверждения.
        </span>
      </div>
    </Screen>
  );
}
