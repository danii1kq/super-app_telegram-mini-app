import { useNavigate } from 'react-router-dom';
import { useConversations } from './api';
import { AppHeader, Avatar, Banner, Card, EmptyState, Hairline, Row, Screen, Skeleton } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { useSession } from '@/app/session';

/** Conversation list with unread counts. */
export function ChatsPage() {
  const t = useT();
  const navigate = useNavigate();
  const { status } = useSession();
  const conversations = useConversations(status === 'ready');

  return (
    <Screen header={<AppHeader title={t('chats')} leadingIcon="chat_bubble" />}>
      {conversations.isLoading ? (
        <div className="flex flex-col gap-2 pt-2">
          <Skeleton className="h-14 w-full" />
          <Skeleton className="h-14 w-full" />
        </div>
      ) : null}

      {conversations.isError ? (
        <div className="pt-2">
          <Banner tone="danger" icon="error" title={t('error')}>
            Проверьте соединение и повторите.
          </Banner>
        </div>
      ) : null}

      {conversations.data && conversations.data.length === 0 ? (
        <EmptyState icon="chat_bubble" title={t('empty')} text="Диалоги появятся, когда вы начнёте переписку." />
      ) : null}

      {conversations.data && conversations.data.length > 0 ? (
        <div className="pt-2">
          <Card>
            {conversations.data.map((conversation, index) => (
              <div key={conversation.id}>
                {index > 0 ? <Hairline /> : null}
                <Row
                  title={conversation.title || (conversation.kind === 'group' ? 'Групповой чат' : 'Диалог')}
                  subtitle={conversation.unread > 0 ? `${conversation.unread} новых` : 'Нет новых сообщений'}
                  onClick={() => navigate(`/chats/${conversation.id}`)}
                  trailing={
                    conversation.unread > 0 ? (
                      <span className="flex size-6 items-center justify-center rounded-full bg-accent text-[12px] font-semibold text-btn-text">
                        {conversation.unread > 99 ? '99+' : conversation.unread}
                      </span>
                    ) : (
                      <Avatar name={conversation.title || 'Ч'} size={32} />
                    )
                  }
                />
              </div>
            ))}
          </Card>
        </div>
      ) : null}
    </Screen>
  );
}
