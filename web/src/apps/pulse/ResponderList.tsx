import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { usePulseResponders } from './api';
import { useOpenDirect } from '@/core/chat/mutations';
import { Avatar, Icon, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';

/**
 * ResponderList shows who answered the author's own status.
 *
 * The audience is private to the author, so this list never renders for anyone
 * else: the server refuses it and the caller only mounts it on own statuses.
 * Each row opens the one-to-one dialog, which is the whole point of a status:
 * someone said "I am free", and the next step is a conversation.
 */
export function ResponderList({statusId}:{statusId:string}) {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const [open, setOpen] = useState(false);
  const responders = usePulseResponders(statusId, open);
  const openDirect = useOpenDirect();

  const startChat = (userId: string) => {
    openDirect.mutate(userId, {
      onSuccess: (conversation) => navigate(`/chats/${conversation.id}`),
      onError: () => feedback(t('error'), 'error'),
    });
  };

  return <div className="pt-2">
    <button
      type="button"
      onClick={() => setOpen((value) => !value)}
      className="flex items-center gap-1 text-body-secondary text-accent"
    >
      <Icon name={open ? 'expand_less' : 'expand_more'} size={18} />
      {open ? t('pulseHideResponders') : t('pulseShowResponders')}
    </button>

    {open ? (
      <div className="flex flex-col gap-1 pt-2">
        {responders.isLoading ? <Skeleton className="h-12 w-full" /> : null}
        {responders.data && responders.data.length === 0 ? (
          <span className="text-caption text-ink-2">{t('pulseNoResponders')}</span>
        ) : null}
        {responders.data?.map((person) => (
          <div key={person.user_id} className="flex items-center gap-2 rounded-xl bg-neutral-soft/60 px-3 py-2">
            <Avatar name={person.name} size={30} />
            <div className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-body-secondary-strong text-ink">{person.name}</span>
              {person.username ? (
                <span className="truncate text-caption text-ink-2">@{person.username}</span>
              ) : null}
            </div>
            <button
              type="button"
              disabled={openDirect.isPending}
              onClick={() => startChat(person.user_id)}
              className="h-8 rounded-lg bg-accent-soft px-3 text-caption font-semibold text-accent"
            >
              {t('pulseWrite')}
            </button>
          </div>
        ))}
      </div>
    ) : null}
  </div>;
}
