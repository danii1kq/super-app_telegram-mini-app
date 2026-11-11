import { AppHeader, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { Placeholder } from '@/shared/ui/Placeholder';

/** Conversation list. */
export function ChatsPage() {
  const t = useT();
  return (
    <Screen header={<AppHeader title={t('chats')} leadingIcon="chat_bubble" />}>
      <Placeholder title={t('chats')} icon="chat_bubble" text="Диалоги появятся после подключения API." />
    </Screen>
  );
}
