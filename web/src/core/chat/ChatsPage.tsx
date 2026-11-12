import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useConversations } from './api';
import { useOpenDirect } from './mutations';
import { usePeople } from '@/core/social/api';
import { AppHeader, Avatar, Banner, Card, EmptyState, Hairline, Icon, PrimaryButton, Screen, SecondaryButton, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useI18n, useT } from '@/shared/i18n';
import { formatDateTime } from '@/shared/i18n/date';
import { useSession } from '@/app/session';

export function ChatsPage() {
  const t = useT();
  const { lang } = useI18n();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { status, me } = useSession();
  const [picking, setPicking] = useState(false);
  const [search, setSearch] = useState('');
  const conversations = useConversations(status === 'ready');
  const people = usePeople(status === 'ready' && picking, search.trim());
  const openDirect = useOpenDirect();
  const term = search.trim().toLocaleLowerCase();
  const matches = (value: string) => value.toLocaleLowerCase().includes(term);
  const shownChats = conversations.data?.filter((chat) => matches(chat.title));
  const shownPeople = people.data?.filter((person) => matches(`${person.public_name} ${person.username ?? ''}`));

  const startWith = (userId: string) => {
    if (openDirect.isPending) return;
    openDirect.mutate(userId, {
      onSuccess: (conversation) => navigate(`/chats/${conversation.id}`),
      onError: () => feedback(t('chatOpenFailed'), 'error'),
    });
  };

  return (
    <Screen header={<AppHeader title={t('chats')} leadingIcon="chat_bubble" trailing={
      <button type="button" aria-label={t('refresh')} disabled={status !== 'ready' || (picking ? people.isFetching : conversations.isFetching)} onClick={() => void (picking ? people.refetch() : conversations.refetch())} className="flex size-11 items-center justify-center text-accent disabled:opacity-50">
        <Icon name="refresh" size={22} />
      </button>
    } />} bottomSpace>
      <div className="flex flex-col gap-3 pt-2">
        <PrimaryButton full disabled={status !== 'ready'} onClick={() => { setPicking(!picking); setSearch(''); }}>
          <Icon name={picking ? 'close' : 'person_add'} size={20} />
          {t(picking ? 'cancel' : 'newChat')}
        </PrimaryButton>
        <SecondaryButton full disabled={status !== 'ready'} onClick={() => navigate('/chats/new-group')}>
          <Icon name="group_add" size={20} />{t('chatNewGroup')}
        </SecondaryButton>
        <label className="flex items-center gap-2 rounded-xl bg-surface px-3 text-ink-2">
          <Icon name="search" size={22} />
          <input value={search} onChange={(event) => setSearch(event.target.value)} aria-label={t(picking ? 'peopleSearch' : 'searchChats')} placeholder={t(picking ? 'peopleSearch' : 'searchChats')} className="h-12 min-w-0 flex-1 bg-transparent text-body-lg text-ink outline-none placeholder:text-hint" />
        </label>
        {(picking ? people.isLoading : conversations.isLoading) ? <Skeleton className="h-28 w-full" /> : null}
        {(picking ? people.isError : conversations.isError) ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            <button type="button" className="min-h-11 text-accent" onClick={() => void (picking ? people.refetch() : conversations.refetch())}>{t('retry')}</button>
          </Banner>
        ) : null}
        {picking ? (
          <Card>
            {shownPeople?.map((person, index) => (
              <div key={person.id}>
                {index ? <Hairline /> : null}
                <button type="button" disabled={openDirect.isPending} onClick={() => startWith(person.id)} className="flex min-h-16 w-full items-center gap-3 px-4 py-3 text-left disabled:opacity-50">
                  <Avatar name={person.public_name} size={40} />
                  <span className="min-w-0 flex-1">
                    <span className="block break-words text-body-strong text-ink">{person.public_name}</span>
                    {person.username ? <span className="text-caption text-ink-2">@{person.username}</span> : null}
                  </span>
                  <Icon name="chevron_right" size={20} className="text-outline-variant" />
                </button>
              </div>
            ))}
            {shownPeople?.length === 0 ? <EmptyState icon="person_search" title={t('empty')} text={t(term ? 'noSearchResults' : 'peopleEmpty')} /> : null}
          </Card>
        ) : (
          <Card>
            {shownChats?.map((conversation, index) => {
              const latest = conversation.last_message;
              const preview = latest ? `${latest.sender_id === me?.id ? `${t('you')}: ` : ''}${latest.text || (latest.attachment?.kind === 'photo' ? t('chatPhoto') : latest.attachment?.name) || t('chatEmptyMessages')}` : t('chatEmptyMessages');
              return (
                <div key={conversation.id}>
                  {index ? <Hairline /> : null}
                  <button type="button" onClick={() => navigate(`/chats/${conversation.id}`)} className="flex min-h-20 w-full items-center gap-3 px-4 py-3 text-left transition active:bg-neutral-soft/60">
                    <Avatar name={conversation.title || t('chat')} size={44} />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-baseline gap-2">
                        <span className="min-w-0 flex-1 truncate text-body-strong text-ink">{conversation.title || t('chat')}</span>
                        {latest ? <time dateTime={latest.created_at} className="shrink-0 text-[11px] text-ink-2">{formatDateTime(latest.created_at, lang)}</time> : null}
                      </span>
                      <span className="mt-1 flex items-center gap-2">
                        <span className="min-w-0 flex-1 truncate text-body-secondary text-ink-2">{preview}</span>
                        {conversation.unread > 0 ? <span aria-label={`${conversation.unread} ${t('newMessages')}`} className="flex min-w-6 shrink-0 items-center justify-center rounded-full bg-accent px-1.5 text-caption font-semibold text-btn-text">{conversation.unread > 99 ? '99+' : conversation.unread}</span> : null}
                      </span>
                    </span>
                  </button>
                </div>
              );
            })}
            {shownChats?.length === 0 ? <EmptyState icon={term ? 'search_off' : 'chat_bubble'} title={t('empty')} text={t(term ? 'noSearchResults' : 'chatsEmpty')} /> : null}
          </Card>
        )}
      </div>
    </Screen>
  );
}
