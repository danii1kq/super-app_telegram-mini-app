import { AppHeader, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { Placeholder } from '@/shared/ui/Placeholder';

/** SruNal feed. The social API lands in feat/core-social. */
export function FeedPage() {
  const t = useT();
  return (
    <Screen header={<AppHeader title="SruNal" leadingIcon="dynamic_feed" />}>
      <Placeholder title={t('feed')} icon="dynamic_feed" text="Лента появится после подключения API." />
    </Screen>
  );
}
