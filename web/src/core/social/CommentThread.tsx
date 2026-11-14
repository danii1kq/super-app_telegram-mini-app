import { useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { useComments } from './api';
import { useCreateComment } from './mutations';
import { Banner, PrimaryButton, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useI18n, useT } from '@/shared/i18n';
import { formatDateTime } from '@/shared/i18n/date';

export function CommentThread({ postId, enabled }: { postId: string; enabled: boolean }) {
  const t = useT();
  const { lang } = useI18n();
  const feedback = useFeedback();
  const comments = useComments(postId, enabled);
  const create = useCreateComment(postId);
  const [draft, setDraft] = useState('');
  const submitting = useRef(false);

  const submit = () => {
    const body = draft.trim();
    if (!body || submitting.current) return;
    submitting.current = true;
    create.mutate(body, {
      onSuccess: () => { setDraft(''); feedback(t('commentSent')); },
      onError: () => feedback(t('commentFailed'), 'error'),
      onSettled: () => { submitting.current = false; },
    });
  };

  return (
    <section id={`comments-${postId}`} aria-label={t('commentsTitle')} className="mt-3 scroll-mt-20 border-t border-line pt-3">
      <h2 className="pb-3 text-body-strong text-ink">{t('commentsTitle')}</h2>
      <textarea value={draft} onChange={(event) => setDraft(event.target.value)} disabled={create.isPending}
        placeholder={t('commentPlaceholder')} aria-label={t('commentPlaceholder')} rows={2} maxLength={2000}
        className="w-full resize-y rounded-xl bg-neutral-soft/60 p-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent placeholder:text-hint disabled:opacity-60" />
      <div className="flex justify-end pt-2"><PrimaryButton onClick={submit} disabled={!draft.trim()} progress={create.isPending}>{t('send')}</PrimaryButton></div>
      {comments.isLoading ? <Skeleton className="mt-3 h-16 w-full" /> : null}
      {comments.isError && !comments.isFetchNextPageError ? <div className="pt-3"><Banner tone="danger" icon="error" title={t('commentLoadFailed')}
        action={<button type="button" onClick={() => void comments.refetch()} className="min-h-11 font-semibold">{t('retry')}</button>} /></div> : null}
      {comments.data?.length === 0 ? <p className="py-3 text-caption text-ink-2">{t('commentEmpty')}</p> : null}
      <div className="flex flex-col gap-3 pt-3">
        {comments.data?.map((comment) => <div key={comment.id} className="rounded-xl bg-neutral-soft/40 p-3">
          <div className="flex flex-wrap items-baseline justify-between gap-x-2 gap-y-1">
            <Link to={`/people/${comment.author_id}`} className="min-h-11 break-words py-2 text-body-secondary-strong text-ink">{comment.author_name}</Link>
            <time dateTime={comment.created_at} className="text-caption text-ink-2">{formatDateTime(comment.created_at, lang)}</time>
          </div>
          <p className="whitespace-pre-wrap break-words pt-1 text-body-lg text-ink [overflow-wrap:anywhere]">{comment.body}</p>
        </div>)}
      </div>
      {comments.isFetchNextPageError ? <p role="alert" className="pt-2 text-caption text-danger">{t('commentLoadFailed')}</p> : null}
      {comments.hasNextPage ? <div className="pt-3"><PrimaryButton full onClick={() => void comments.fetchNextPage()}
        disabled={comments.isFetching} progress={comments.isFetchingNextPage}>{comments.isFetchNextPageError ? t('retry') : t('loadMore')}</PrimaryButton></div> : null}
    </section>
  );
}
