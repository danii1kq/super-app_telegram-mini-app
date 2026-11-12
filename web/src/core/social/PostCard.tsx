import { useState } from 'react';
import type { Post } from './api';
import { useDeletePost, useToggleReaction } from './mutations';
import { CommentThread } from './CommentThread';
import { Avatar, Card, Icon } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useI18n, useT } from '@/shared/i18n';
import { formatDateTime } from '@/shared/i18n/date';
import { useSession } from '@/app/session';
import { showConfirm } from '@/shared/telegram/sdk';

export function PostCard({ post }: { post: Post }) {
  const t = useT();
  const { lang } = useI18n();
  const { me } = useSession();
  const feedback = useFeedback();
  const reaction = useToggleReaction();
  const deletion = useDeletePost();
  const [commentsOpen, setCommentsOpen] = useState(false);
  const [confirming, setConfirming] = useState(false);

  const remove = async () => {
    if (confirming || deletion.isPending) return;
    setConfirming(true);
    try {
      if (await showConfirm(t('confirmDeletePost'))) {
        deletion.mutate(post.id, {
          onSuccess: () => feedback(t('postDeleted')),
          onError: () => feedback(t('postDeleteFailed'), 'error'),
        });
      }
    } finally { setConfirming(false); }
  };

  return (
    <Card className="p-4">
      <div className="flex items-center gap-2">
        <Avatar name={post.author_name} size={36} />
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="break-words text-body-strong text-ink">{post.author_name}</span>
          <time dateTime={post.created_at} className="text-caption text-ink-2">{formatDateTime(post.created_at, lang)}</time>
        </div>
        {post.author_id === me?.id ? <button type="button" onClick={() => void remove()}
          disabled={confirming || deletion.isPending} aria-label={t('delete')}
          className="-mr-2 flex size-11 shrink-0 items-center justify-center text-ink-2 disabled:opacity-50">
          <Icon name="delete_outline" size={20} />
        </button> : null}
      </div>
      <p className="whitespace-pre-wrap break-words pt-3 text-body-lg text-ink [overflow-wrap:anywhere]">{post.body}</p>
      <div className="-mb-1 -ml-2 flex items-center gap-2 pt-2">
        <button type="button" disabled={reaction.isPending} aria-pressed={post.liked_by_me}
          aria-label={`${post.liked_by_me ? t('unlike') : t('like')}: ${post.likes}`}
          onClick={() => reaction.mutate(post.id, { onError: () => feedback(t('saveFailed'), 'error') })}
          className={`flex min-h-11 min-w-11 items-center gap-1.5 rounded-xl px-2 text-body-secondary disabled:opacity-50 ${post.liked_by_me ? 'text-accent' : 'text-ink-2'}`}>
          <Icon name="favorite" size={20} filled={post.liked_by_me} />{post.likes}
        </button>
        <button type="button" aria-expanded={commentsOpen} aria-controls={`comments-${post.id}`}
          onClick={() => setCommentsOpen((value) => !value)}
          className={`flex min-h-11 min-w-11 items-center gap-1.5 rounded-xl px-2 text-body-secondary ${commentsOpen ? 'text-accent' : 'text-ink-2'}`}>
          <Icon name="chat_bubble" size={20} /><span>{t('commentsTitle')} · {post.comments}</span>
        </button>
      </div>
      <div hidden={!commentsOpen}><CommentThread postId={post.id} enabled={commentsOpen} /></div>
    </Card>
  );
}
