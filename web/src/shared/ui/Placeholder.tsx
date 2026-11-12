import { AppHeader, EmptyState, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';

/** Placeholder screen used until the section's API lands. */
export function Placeholder({
  title,
  icon,
  text,
}: {
  title: string;
  icon: string;
  text?: string;
}) {
  const t = useT();
  return (
    <Screen header={<AppHeader title={title} leadingIcon={icon} />}>
      <EmptyState icon={icon} title={t('empty')} text={text} />
    </Screen>
  );
}
