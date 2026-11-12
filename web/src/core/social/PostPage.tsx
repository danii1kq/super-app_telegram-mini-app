import { useEffect } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { usePost } from './api';
import { PostCard } from './PostCard';
import { CommentThread } from './CommentThread';
import { useSession } from '@/app/session';
import { useT } from '@/shared/i18n';
import { AppHeader, Banner, Screen, Skeleton } from '@/shared/ui';

export function PostPage() {
  const { id = '' } = useParams();
  const t = useT();
  const navigate = useNavigate();
  const { hash } = useLocation();
  const { status } = useSession();
  const post = usePost(id, status === 'ready');

  useEffect(() => {
    if (!post.data || !hash) return;
    const frame = requestAnimationFrame(() => document.getElementById(`comments-${id}`)?.scrollIntoView({ block: 'start' }));
    return () => cancelAnimationFrame(frame);
  }, [id, hash, post.data?.id]);

  return <Screen header={<AppHeader title={t('postTitle')} onBack={() => navigate('/feed')} />} bottomSpace>
    <div className="pt-3">
      {status === 'outside' ? <Banner tone="warning" icon="info" title={t('openInTelegram')} /> : null}
      {post.isLoading ? <Skeleton className="h-48 w-full" /> : null}
      {post.isError ? <Banner tone="danger" icon="error" title={t('postUnavailable')} action={<button type="button" onClick={() => void post.refetch()} className="min-h-11 font-semibold">{t('retry')}</button>} /> : null}
      {post.data && !post.isError && status === 'ready' ? <>
        <PostCard key={post.data.id} post={post.data} detail onDeleted={() => navigate('/feed', { replace: true })} />
        <CommentThread key={id} postId={id} enabled />
      </> : null}
    </div>
  </Screen>;
}
