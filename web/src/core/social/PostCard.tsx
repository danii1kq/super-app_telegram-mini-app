import { useState } from 'react';
import { Link } from 'react-router-dom';
import type { Post } from './api';
import { useDeletePost, useToggleReaction } from './mutations';
import { usePostView } from './usePostView';
import { Avatar, Card, Icon } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useI18n, useT } from '@/shared/i18n';
import { formatDateTime } from '@/shared/i18n/date';
import { useSession } from '@/app/session';
import { showConfirm } from '@/shared/telegram/sdk';

export function PostCard({ post, detail = false, onDeleted }: { post: Post; detail?: boolean; onDeleted?: () => void }) {
  const t = useT();
  const { lang } = useI18n();
  const { me, status } = useSession();
  const feedback = useFeedback();
  const reaction = useToggleReaction();
  const deletion = useDeletePost(() => { feedback(t('postDeleted')); onDeleted?.(); });
  const [confirming, setConfirming] = useState(false);
  const view = usePostView(post.id, me?.id ?? '', status === 'ready' && post.author_id !== me?.id);

  const remove = async () => {
    if (confirming || deletion.isPending) return;
    setConfirming(true);
    try {
      if (await showConfirm(t('confirmDeletePost'))) {
        deletion.mutate(post.id, {
          onError: () => feedback(t('postDeleteFailed'), 'error'),
        });
      }
    } finally { setConfirming(false); }
  };

  return (
    <article ref={view} className="relative">
    <Card className="p-4">
      {!detail ? <Link to={`/posts/${post.id}`} aria-label={`${t('postOpen')}: ${post.author_name}`}
        className="absolute inset-0 z-0 rounded-[14px] outline-none focus-visible:ring-2 focus-visible:ring-accent" /> : null}
      <div className="flex items-center gap-2">
        <Link to={`/people/${post.author_id}`} className="relative z-10 flex min-w-0 flex-1 items-center gap-2 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-accent">
        <Avatar name={post.author_name} size={36} />
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="break-words text-body-strong text-ink">{post.author_name}</span>
          <time dateTime={post.created_at} className="text-caption text-ink-2">{formatDateTime(post.created_at, lang)}</time>
        </div>
        </Link>
        {post.author_id === me?.id ? <button type="button" onClick={() => void remove()}
          disabled={confirming || deletion.isPending} aria-label={t('delete')}
          className="relative z-10 -mr-2 flex size-11 shrink-0 items-center justify-center text-ink-2 disabled:opacity-50">
          <Icon name="delete_outline" size={20} />
        </button> : null}
      </div>
      <p className="whitespace-pre-wrap break-words pt-3 text-body-lg text-ink [overflow-wrap:anywhere]">{post.body}</p>
      <div className="-mb-1 -ml-2 flex items-center gap-2 pt-2">
        <button type="button" disabled={reaction.isPending} aria-pressed={post.liked_by_me}
          aria-label={`${post.liked_by_me ? t('unlike') : t('like')}: ${post.likes}`}
          onClick={() => reaction.mutate(post.id, { onError: () => feedback(t('saveFailed'), 'error') })}
          className={`relative z-10 flex min-h-11 min-w-11 items-center gap-1.5 rounded-xl px-2 text-body-secondary disabled:opacity-50 ${post.liked_by_me ? 'text-accent' : 'text-ink-2'}`}>
          <Icon name="favorite" size={20} filled={post.liked_by_me} />{post.likes}
        </button>
        <Link to={`${detail ? '' : `/posts/${post.id}`}#comments-${post.id}`} aria-label={`${t('commentsTitle')}: ${post.comments}`}
          className="relative z-10 flex min-h-11 min-w-11 items-center gap-1.5 rounded-xl px-2 text-body-secondary text-ink-2">
          <Icon name="chat_bubble" size={20} />{post.comments}
        </Link>
        <span className="ml-auto flex items-center gap-1 text-caption tabular-nums text-ink-2" aria-label={`${t('postViews')}: ${post.views}`} title={t('postViews')}>
          <Icon name="visibility" size={17} />{post.views}
        </span>
      </div>
    </Card>
    </article>
  );
}
