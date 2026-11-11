import { AppHeader, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { Placeholder } from '@/shared/ui/Placeholder';

/** Profile and settings. */
export function ProfilePage() {
  const t = useT();
  return (
    <Screen header={<AppHeader title={t('profile')} leadingIcon="person" />}>
      <Placeholder title={t('profile')} icon="person" text="Профиль появится после подключения API." />
    </Screen>
  );
}
