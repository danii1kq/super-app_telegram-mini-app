import { useRef, useState } from 'react';
import { useFeed } from './api';
import { useCreatePost } from './mutations';
import { PostCard } from './PostCard';
import { AppHeader, Banner, Card, EmptyState, Icon, PrimaryButton, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';
import { useSession } from '@/app/session';

export function FeedPage() {
  const t = useT();
  const feedback = useFeedback();
  const { status } = useSession();
  const ready = status === 'ready';
  const feed = useFeed(ready);
  const createPost = useCreatePost();
  const [draft, setDraft] = useState('');
  const [composerOpen, setComposerOpen] = useState(false);
  const submitting = useRef(false);

  const submit = () => {
    const body = draft.trim();
    if (!body || !ready || submitting.current) return;
    submitting.current = true;
    createPost.mutate({ body, audience: 'public' }, {
      onSuccess: () => { setDraft(''); setComposerOpen(false); feedback(t('published')); },
      onError: () => feedback(t('publishFailed'), 'error'),
      onSettled: () => { submitting.current = false; },
    });
  };

  return (
    <Screen header={<AppHeader title="SruNal" leadingIcon="dynamic_feed" trailing={
      <button type="button" onClick={() => void feed.refetch()} disabled={!ready || feed.isFetching}
        aria-label={t('refresh')} className="flex size-11 items-center justify-center text-accent disabled:opacity-40">
        <Icon name="refresh" size={22} />
      </button>
    } />}>
      {status === 'outside' ? <div className="pt-2"><Banner tone="warning" icon="info" title={t('openInTelegram')}>{t('feedOutside')}</Banner></div> : null}
      <div className="pt-2">
        {composerOpen ? (
          <Card className="p-3">
            <textarea value={draft} onChange={(event) => setDraft(event.target.value)} autoFocus
              placeholder={t('postPlaceholder')} aria-label={t('postPlaceholder')} rows={4} maxLength={4000}
              disabled={createPost.isPending}
              className="w-full resize-y rounded-xl bg-neutral-soft/60 p-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent placeholder:text-hint disabled:opacity-60" />
            <p className="px-1 pt-2 text-caption text-ink-2">{t('publicPostHint')}</p>
            <div className="flex items-center justify-between gap-2 pt-2">
              <button type="button" disabled={createPost.isPending} onClick={() => setComposerOpen(false)}
                className="min-h-11 px-2 text-body-secondary text-ink-2 disabled:opacity-50">{t('cancel')}</button>
              <span className="text-caption tabular-nums text-ink-2">{draft.length}/4000</span>
              <PrimaryButton onClick={submit} disabled={!draft.trim() || !ready} progress={createPost.isPending}>{t('send')}</PrimaryButton>
            </div>
          </Card>
        ) : (
          <button type="button" onClick={() => setComposerOpen(true)} disabled={!ready}
            className="flex min-h-14 w-full items-center gap-3 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint disabled:opacity-50">
            <Icon name="edit" size={22} className="text-accent" />{t('writePost')}
          </button>
        )}
      </div>
      <div className="flex flex-col gap-3 pt-3">
        {feed.isLoading ? <><Skeleton className="h-28 w-full" /><Skeleton className="h-28 w-full" /></> : null}
        {feed.isError && !feed.isFetchNextPageError ? (
          <Banner tone="danger" icon="error" title={t('error')} action={
            <button type="button" onClick={() => void feed.refetch()} className="min-h-11 font-semibold">{t('retry')}</button>
          }>{t('retryConnection')}</Banner>
        ) : null}
        {ready && feed.data?.length === 0 ? <EmptyState icon="dynamic_feed" title={t('empty')} text={t('feedEmpty')} /> : null}
        {feed.data?.map((post) => <PostCard key={post.id} post={post} />)}
        {feed.isFetchNextPageError ? <p role="alert" className="text-body-secondary text-danger">{t('retryConnection')}</p> : null}
        {feed.hasNextPage ? <PrimaryButton full onClick={() => void feed.fetchNextPage()}
          disabled={feed.isFetching} progress={feed.isFetchingNextPage}>{feed.isFetchNextPageError ? t('retry') : t('loadMore')}</PrimaryButton> : null}
      </div>
    </Screen>
  );
}
