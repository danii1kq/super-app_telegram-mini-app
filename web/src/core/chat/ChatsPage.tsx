import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useConversations } from './api';
import { useOpenDirect } from './mutations';
import { usePeople } from '@/core/social/api';
import {
  AppHeader,
  Avatar,
  Banner,
  Card,
  EmptyState,
  Hairline,
  Icon,
  PrimaryButton,
  Row,
  Screen,
  Skeleton,
} from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { useSession } from '@/app/session';

/** Conversation list with unread counts and a way to start a new dialog. */
export function ChatsPage() {
  const t = useT();
  const navigate = useNavigate();
  const { status } = useSession();
  const conversations = useConversations(status === 'ready');
  const people = usePeople(status === 'ready');
  const openDirect = useOpenDirect();
  const [picking, setPicking] = useState(false);

  const startWith = (userId: string) => {
    openDirect.mutate(userId, {
      onSuccess: (conversation) => {
        setPicking(false);
        navigate(`/chats/${conversation.id}`);
      },
    });
  };

  return (
    <Screen header={<AppHeader title={t('chats')} leadingIcon="chat_bubble" />}>
      {picking ? (
        <Card className="mt-2">
          <div className="flex items-center justify-between px-4 pt-3">
            <span className="text-body-strong text-ink">{t('newChat')}</span>
            <button type="button" onClick={() => setPicking(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
          </div>
          {people.isLoading ? (
            <div className="p-4">
              <Skeleton className="h-12 w-full" />
            </div>
          ) : null}
          {people.data && people.data.length === 0 ? (
            <div className="px-4 py-3 text-caption text-ink-2">{t('peopleEmpty')}</div>
          ) : null}
          {people.data?.map((person, index) => (
            <div key={person.id}>
              {index > 0 ? <Hairline /> : null}
              <Row
                title={person.public_name}
                subtitle={person.username ? `@${person.username}` : undefined}
                onClick={() => startWith(person.id)}
                trailing={<Icon name="chevron_right" size={20} className="text-outline-variant" />}
              />
            </div>
          ))}
        </Card>
      ) : (
        <div className="pt-2">
          <PrimaryButton onClick={() => setPicking(true)} full>
            <Icon name="person_add" size={20} />
            {t('newChat')}
          </PrimaryButton>
        </div>
      )}

      {conversations.isLoading ? (
        <div className="flex flex-col gap-2 pt-3">
          <Skeleton className="h-14 w-full" />
          <Skeleton className="h-14 w-full" />
        </div>
      ) : null}

      {conversations.isError ? (
        <div className="pt-3">
          <Banner tone="danger" icon="error" title={t('error')}>
            {t('retryConnection')}
          </Banner>
        </div>
      ) : null}

      {conversations.data && conversations.data.length === 0 ? (
        <EmptyState icon="chat_bubble" title={t('empty')} text={t('chatsEmpty')} />
      ) : null}

      {conversations.data && conversations.data.length > 0 ? (
        <div className="pt-3">
          <Card>
            {conversations.data.map((conversation, index) => (
              <div key={conversation.id}>
                {index > 0 ? <Hairline /> : null}
                <Row
                  title={conversation.title || (conversation.kind === 'group' ? t('chat') : t('chat'))}
                  subtitle={
                    conversation.unread > 0
                      ? `${conversation.unread} ${t('newMessages')}`
                      : t('noMessages')
                  }
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
