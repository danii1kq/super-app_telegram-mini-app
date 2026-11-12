import { useNavigate, useParams } from 'react-router-dom';
import { useAuthorPosts, useUserProfile } from '@/core/social/api';
import { useSetFollowing } from '@/core/social/mutations';
import { PostCard } from '@/core/social/PostCard';
import { useSession } from '@/app/session';
import { useI18n, useT } from '@/shared/i18n';
import { formatDayMonth } from '@/shared/i18n/date';
import { openTelegramLink } from '@/shared/telegram/sdk';
import { AppHeader, Avatar, Banner, Card, EmptyState, Icon, PrimaryButton, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';

export function UserProfilePage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const t = useT();
  const { lang } = useI18n();
  const { status } = useSession();
  const feedback = useFeedback();
  const profile = useUserProfile(id, status === 'ready');
  const posts = useAuthorPosts(id, status === 'ready' && Boolean(profile.data) && !profile.isError);
  const follow = useSetFollowing(id);
  const person = profile.data;

  return <Screen header={<AppHeader title={t('profile')} onBack={() => navigate('/feed')} />} bottomSpace>
    <div className="flex flex-col gap-3 pt-3">
      {status === 'outside' ? <Banner tone="warning" icon="info" title={t('openInTelegram')} /> : null}
      {profile.isLoading ? <Skeleton className="h-56 w-full" /> : null}
      {profile.isError ? <Banner tone="danger" icon="error" title={t('profileUnavailable')} action={<button type="button" onClick={() => void profile.refetch()} className="min-h-11 font-semibold">{t('retry')}</button>} /> : null}
      {person && !profile.isError && status === 'ready' ? <>
        <Card className="gap-4 p-4">
          <div className="flex items-center gap-3"><Avatar name={person.public_name} size={64} /><div className="min-w-0 flex-1">
            <h2 className="break-words text-headline-md text-ink [overflow-wrap:anywhere]">{person.public_name}</h2>
            {person.username ? <p className="break-words text-body-secondary text-ink-2">@{person.username}</p> : null}
            <p className="pt-1 text-caption text-ink-2">{t('profileJoined')} {formatDayMonth(person.joined_at, lang)} {new Date(person.joined_at).getFullYear()}</p>
          </div></div>
          <div className="grid grid-cols-3 gap-2 text-center">{[
            [person.posts_count, t('profilePosts')], [person.followers, t('profileFollowers')], [person.following_count, t('profileFollowing')],
          ].map(([count, label]) => <div key={label} className="min-w-0"><span className="block text-headline-md tabular-nums text-ink">{count}</span><span className="break-words text-caption text-ink-2">{label}</span></div>)}</div>
          <div className="rounded-xl bg-neutral-soft/60 p-3">
            <p className="flex items-center gap-2 text-body-strong text-ink"><Icon name={person.is_friend ? 'group' : 'person'} size={20} />{t(person.is_me ? 'profileOwn' : person.is_friend ? 'profileFriends' : person.following ? 'profileYouFollow' : person.follows_you ? 'profileFollowsYou' : 'profileNotFriends')}</p>
            {!person.is_me ? <p className="pt-1 text-caption text-ink-2">{t('profileMutualHint')}</p> : null}
          </div>
          {!person.is_me ? <>
            <PrimaryButton full progress={follow.isPending} onClick={() => follow.mutate(!person.following, { onError: () => feedback(t('saveFailed'), 'error') })}>{t(person.following ? 'unfollow' : 'follow')}</PrimaryButton>
            <button type="button" disabled={!person.telegram_url} onClick={() => { if (person.telegram_url) openTelegramLink(person.telegram_url); }} className="flex min-h-12 items-center justify-center gap-2 rounded-xl bg-accent-soft px-3 text-body-strong text-accent disabled:opacity-50"><Icon name="send" size={20} />{t('profileTelegram')}</button>
            {!person.telegram_url ? <p className="text-caption text-ink-2">{t('profileTelegramUnavailable')}</p> : null}
          </> : null}
        </Card>
        <h2 className="px-1 pt-1 text-body-strong text-ink">{t('profilePosts')}</h2>
        {posts.isLoading ? <Skeleton className="h-32 w-full" /> : null}
        {posts.isError && !posts.isFetchNextPageError ? <Banner tone="danger" icon="error" title={t('error')} action={<button type="button" onClick={() => void posts.refetch()} className="min-h-11 font-semibold">{t('retry')}</button>} /> : null}
        {posts.data?.length === 0 ? <EmptyState icon="dynamic_feed" title={t('empty')} /> : null}
        {posts.data?.map((post) => <PostCard key={post.id} post={post} />)}
        {posts.hasNextPage ? <PrimaryButton full onClick={() => void posts.fetchNextPage()} disabled={posts.isFetching} progress={posts.isFetchingNextPage}>{t(posts.isFetchNextPageError ? 'retry' : 'loadMore')}</PrimaryButton> : null}
      </> : null}
    </div>
  </Screen>;
}
