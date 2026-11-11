import { useNavigate } from 'react-router-dom';
import { AppHeader, Screen } from '@/shared/ui';
import { Placeholder } from '@/shared/ui/Placeholder';

/** Single conversation. */
export function ChatPage() {
  const navigate = useNavigate();
  return (
    <Screen header={<AppHeader title="Чат" onBack={() => navigate('/chats')} />}>
      <Placeholder title="Чат" icon="chat_bubble" text="История сообщений появится после подключения API." />
    </Screen>
  );
}
