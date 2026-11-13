import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  useLike,
  usePass,
  useSaveProfile,
  useTinderDeck,
  useTinderMatches,
  useTinderProfile,
  type TinderCard,
} from '../api';
import { useOpenDirect } from '@/core/chat/mutations';
import {
  AppHeader,
  Avatar,
  Banner,
  Card,
  Chip,
  EmptyState,
  Icon,
  PrimaryButton,
  Screen,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { useT } from '@/shared/i18n';

/** SruTinder: a profile, a deck, a like, a match, a chat. */
export function TinderPage() {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const profile = useTinderProfile(Boolean(me));
  const deck = useTinderDeck(Boolean(me));
  const matches = useTinderMatches(Boolean(me));
  const saveProfile = useSaveProfile();
  const like = useLike();
  const pass = usePass();
  const openDirect = useOpenDirect();

  const [editing, setEditing] = useState(false);
  const [bio, setBio] = useState('');
  const [university, setUniversity] = useState('');
  const [age, setAge] = useState('');
  const [photo, setPhoto] = useState('');
  const [showMatches, setShowMatches] = useState(false);

  const own = profile.data?.profile ?? null;
  const likesMe = profile.data?.likes_me ?? 0;
  const cards = deck.data ?? [];
  const card = cards[0];

  const startEditing = () => {
    setBio(own?.bio ?? '');
    setUniversity(own?.university ?? '');
    setAge(own?.age ? String(own.age) : '');
    setPhoto(own?.photo_url ?? '');
    setEditing(true);
  };

  const submitProfile = () => {
    saveProfile.mutate(
      {
        bio: bio.trim(),
        university: university.trim(),
        age: Number(age) || 0,
        photo_url: photo.trim(),
        active: true,
      },
      {
        onSuccess: () => {
          setEditing(false);
          feedback(t('tinderSaved'));
        },
        onError: () => feedback(t('tinderSaveFailed'), 'error'),
      },
    );
  };

  const sendLike = (target: TinderCard) => {
    like.mutate(target.user_id, {
      onSuccess: (result) => {
        if (result.matched) feedback(t('tinderMatch'));
      },
      onError: () => feedback(t('error'), 'error'),
    });
  };

  const openChat = (partnerId: string) => {
    openDirect.mutate(partnerId, {
      onSuccess: (conversation) => navigate(`/chats/${conversation.id}`),
      onError: () => feedback(t('error'), 'error'),
    });
  };

  return (
    <Screen header={<AppHeader title="SruTinder" onBack={() => navigate('/apps')} leadingIcon="favorite" />}>
      {!own && !editing ? (
        <Card className="mt-2 p-4">
          <div className="flex flex-col items-center gap-3 py-4 text-center">
            <Icon name="favorite" size={40} className="text-accent" />
            <span className="text-body-strong text-ink">{t('tinderNoProfile')}</span>
            <span className="text-body-secondary text-ink-2">{t('tinderNoProfileHint')}</span>
            <PrimaryButton onClick={startEditing}>{t('tinderCreate')}</PrimaryButton>
          </div>
        </Card>
      ) : null}

      {editing ? (
        <Card className="mt-2 p-3">
          <div className="flex flex-col gap-2">
            <input
              value={photo}
              onChange={(event) => setPhoto(event.target.value)}
              placeholder={t('tinderPhoto')}
              className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
            />
            <textarea
              value={bio}
              onChange={(event) => setBio(event.target.value)}
              placeholder={t('tinderBio')}
              rows={3}
              maxLength={500}
              className="w-full resize-none rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
            />
            <div className="flex gap-2">
              <input
                value={university}
                onChange={(event) => setUniversity(event.target.value)}
                placeholder={t('tinderUniversity')}
                className="flex-1 rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
              />
              <input
                value={age}
                onChange={(event) => setAge(event.target.value.replace(/\D/g, '').slice(0, 3))}
                placeholder={t('tinderAge')}
                inputMode="numeric"
                className="w-20 rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
              />
            </div>
          </div>
          <div className="flex items-center justify-between pt-3">
            <button type="button" onClick={() => setEditing(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
            <PrimaryButton onClick={submitProfile} progress={saveProfile.isPending}>
              {t('save')}
            </PrimaryButton>
          </div>
        </Card>
      ) : null}

      {own && !editing ? (
        <div className="flex items-center gap-2 pt-3">
          <button type="button" onClick={startEditing} className="text-body-secondary text-accent">
            {t('tinderEditProfile')}
          </button>
          <span className="flex-1" />
          {likesMe > 0 ? (
            <Chip tone="success" icon="favorite">
              {likesMe} {t('tinderLikesYou')}
            </Chip>
          ) : null}
          <button
            type="button"
            onClick={() => setShowMatches((value) => !value)}
            className="text-body-secondary text-accent"
          >
            {t('tinderMatches')} {matches.data?.length ?? 0}
          </button>
        </div>
      ) : null}

      {showMatches ? (
        <div className="flex flex-col gap-1 pt-3">
          {matches.data && matches.data.length === 0 ? (
            <span className="text-caption text-ink-2">{t('tinderNoMatches')}</span>
          ) : null}
          {matches.data?.map((match) => (
            <div key={match.id} className="flex items-center gap-2 rounded-xl bg-neutral-soft/60 px-3 py-2">
              <Avatar name={match.name} size={32} />
              <span className="min-w-0 flex-1 truncate text-body-secondary-strong text-ink">{match.name}</span>
              <button
                type="button"
                disabled={openDirect.isPending}
                onClick={() => openChat(match.partner_id)}
                className="h-8 rounded-lg bg-accent-soft px-3 text-caption font-semibold text-accent"
              >
                {t('pulseWrite')}
              </button>
            </div>
          ))}
        </div>
      ) : null}

      {own && !editing && !showMatches ? (
        <div className="pt-3">
          {deck.isLoading ? <Skeleton className="h-80 w-full" /> : null}
          {deck.isError ? (
            <Banner tone="danger" icon="error" title={t('error')}>
              {t('retryConnection')}
            </Banner>
          ) : null}
          {deck.data && cards.length === 0 ? (
            <EmptyState icon="favorite" title={t('tinderDeckEmpty')} text={t('tinderDeckEmptyHint')} />
          ) : null}

          {card ? (
            <Card className="overflow-hidden p-0">
              {card.photo_url ? (
                <img src={card.photo_url} alt={card.name} className="h-72 w-full object-cover" />
              ) : (
                <div className="flex h-72 w-full items-center justify-center bg-neutral-soft">
                  <Avatar name={card.name} size={96} />
                </div>
              )}
              <div className="p-4">
                <div className="flex items-center gap-2">
                  <span className="text-title-3 text-ink">{card.name}</span>
                  {card.age ? <span className="text-body-lg text-ink-2">{card.age}</span> : null}
                  {card.likes_you ? (
                    <Chip tone="success" icon="favorite">
                      {t('tinderLikesYou')}
                    </Chip>
                  ) : null}
                </div>
                {card.university ? <div className="pt-1 text-caption text-ink-2">{card.university}</div> : null}
                {card.bio ? <p className="whitespace-pre-wrap pt-2 text-body-lg text-ink">{card.bio}</p> : null}

                <div className="flex items-center gap-2 pt-4">
                  <button
                    type="button"
                    disabled={pass.isPending || like.isPending}
                    onClick={() => pass.mutate(card.user_id)}
                    className="h-12 flex-1 rounded-xl bg-neutral-soft text-body-secondary-strong text-ink-2"
                  >
                    {t('tinderPass')}
                  </button>
                  <button
                    type="button"
                    disabled={pass.isPending || like.isPending}
                    onClick={() => sendLike(card)}
                    className="h-12 flex-1 rounded-xl bg-accent text-body-secondary-strong text-white"
                  >
                    {t('tinderLike')}
                  </button>
                </div>
                <div className="pt-2 text-center text-caption text-ink-2">
                  {t('tinderLeft')} {cards.length}
                </div>
              </div>
            </Card>
          ) : null}
        </div>
      ) : null}
    </Screen>
  );
}
