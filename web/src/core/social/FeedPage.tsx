import { useState } from 'react';
import { useFeed } from './api';
import { useCreatePost, useToggleReaction } from './mutations';
import { AppHeader, Avatar, Banner, Card, EmptyState, Icon, PrimaryButton, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';
import { useSession } from '@/app/session';

/** SruNal feed: the start screen of the shell. */
export function FeedPage() {
  const t = useT();
  const feedback = useFeedback();
  const { status } = useSession();
  const feed = useFeed(status === 'ready');
  const createPost = useCreatePost();
  const toggleReaction = useToggleReaction();
  const [draft, setDraft] = useState('');
  const [composerOpen, setComposerOpen] = useState(false);

  const submit = () => {
    const body = draft.trim();
    if (!body) return;
    createPost.mutate(
      { body, audience: 'public' },
      {
        onSuccess: () => {
          setDraft('');
          setComposerOpen(false);
          feedback('Опубликовано');
        },
        onError: () => feedback('Не удалось опубликовать', 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title="SruNal" leadingIcon="dynamic_feed" />}>
      {status === 'outside' ? (
        <div className="pt-2">
          <Banner tone="warning" icon="info" title="Откройте через Telegram">
            Лента работает внутри Telegram Mini App: вход подтверждается подписью Telegram.
          </Banner>
        </div>
      ) : null}

      <div className="pt-2">
        {composerOpen ? (
          <Card className="p-3">
            <textarea
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              placeholder="Что нового?"
              rows={3}
              maxLength={4000}
              className="w-full resize-none rounded-xl bg-neutral-soft/60 p-3 text-body-lg text-ink outline-none placeholder:text-hint"
            />
            <div className="flex items-center justify-between pt-2">
              <button
                type="button"
                onClick={() => setComposerOpen(false)}
                className="text-body-secondary text-ink-2"
              >
                {t('cancel')}
              </button>
              <PrimaryButton onClick={submit} disabled={!draft.trim()} progress={createPost.isPending}>
                {t('send')}
              </PrimaryButton>
            </div>
          </Card>
        ) : (
          <button
            type="button"
            onClick={() => setComposerOpen(true)}
            className="flex w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint"
          >
            <Icon name="edit" size={20} className="text-accent" />
            Написать пост
          </button>
        )}
      </div>

      <div className="flex flex-col gap-2 pt-3">
        {feed.isLoading ? (
          <>
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </>
        ) : null}

        {feed.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            Проверьте соединение и повторите.
          </Banner>
        ) : null}

        {feed.data && feed.data.length === 0 ? (
          <EmptyState
            icon="dynamic_feed"
            title={t('empty')}
            text="Подпишитесь на авторов или вступите в сообщество, чтобы увидеть публикации."
          />
        ) : null}

        {feed.data?.map((post) => (
          <Card key={post.id} className="p-4">
            <div className="flex items-center gap-2">
              <Avatar name={post.author_name} size={36} />
              <div className="flex min-w-0 flex-col">
                <span className="truncate text-body-strong text-ink">{post.author_name}</span>
                <span className="text-caption text-ink-2">
                  {new Date(post.created_at).toLocaleString('ru-RU', {
                    day: 'numeric',
                    month: 'short',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </span>
              </div>
            </div>
            <p className="whitespace-pre-wrap pt-2 text-body-lg text-ink">{post.body}</p>
            <div className="flex items-center gap-4 pt-3">
              <button
                type="button"
                onClick={() => toggleReaction.mutate(post.id)}
                className={`flex items-center gap-1 text-body-secondary ${
                  post.liked_by_me ? 'text-accent' : 'text-ink-2'
                }`}
              >
                <Icon name="favorite" size={18} filled={post.liked_by_me} />
                {post.likes}
              </button>
              <span className="flex items-center gap-1 text-body-secondary text-ink-2">
                <Icon name="chat_bubble" size={18} />
                {post.comments}
              </span>
            </div>
          </Card>
        ))}
      </div>
    </Screen>
  );
}
