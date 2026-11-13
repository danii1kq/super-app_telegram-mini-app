import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { minutesLeft, usePulseFeed } from '../api';
import { useCloseStatus, useCreateStatus, useRespond } from '../mutations';
import { ResponderList } from '../ResponderList';
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
import { usePeople } from '@/core/social/api';
import { useT } from '@/shared/i18n';

const DURATIONS = [
  { value: 30, label: '30' },
  { value: 60, label: '60' },
  { value: 120, label: '120' },
] as const;

/** SruPulse: who is free right now. */
export function PulseHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const feed = usePulseFeed(Boolean(me));
  const createStatus = useCreateStatus();
  const respond = useRespond();
  const closeStatus = useCloseStatus();
  const people = usePeople(Boolean(me));

  const [composing, setComposing] = useState(false);
  const [text, setText] = useState('');
  const [minutes, setMinutes] = useState<number>(60);
  const [audience, setAudience] = useState<string[]>([]);

  const toggleAudience = (id: string) => {
    setAudience((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    );
  };

  const submit = () => {
    const value = text.trim();
    if (!value || !me) return;
    // The author always sees their own status; the audience adds the rest.
    const recipients = audience.includes(me.id) ? audience : [me.id, ...audience];
    const expiresAt = new Date(Date.now() + minutes * 60_000).toISOString();
    createStatus.mutate(
      { text: value, expiresAt, audience: recipients },
      {
        onSuccess: () => {
          setText('');
          setAudience([]);
          setComposing(false);
          feedback(t('pulsePublished'));
        },
        onError: () => feedback(t('pulseFailed'), 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title="SruPulse" onBack={() => navigate('/apps')} leadingIcon="bolt" />}>
      {composing ? (
        <Card className="mt-2 p-3">
          <textarea
            value={text}
            onChange={(event) => setText(event.target.value)}
            placeholder={t('pulsePlaceholder')}
            rows={2}
            maxLength={280}
            className="w-full resize-none rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <div className="flex items-center gap-1.5 pt-2">
            {DURATIONS.map((duration) => (
              <button
                key={duration.value}
                type="button"
                onClick={() => setMinutes(duration.value)}
                className={
                  minutes === duration.value
                    ? 'h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                    : 'h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                }
              >
                {duration.label} {t('minutesShort')}
              </button>
            ))}
          </div>

          <div className="pt-3 text-caption text-ink-2">{t('audience')}</div>
          <div className="flex flex-wrap gap-1.5 pt-1">
            {people.data?.map((person) => {
              const picked = audience.includes(person.id);
              return (
                <button
                  key={person.id}
                  type="button"
                  onClick={() => toggleAudience(person.id)}
                  className={
                    picked
                      ? 'h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                      : 'h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                  }
                >
                  {person.public_name}
                </button>
              );
            })}
          </div>

          <div className="flex items-center justify-between pt-3">
            <button type="button" onClick={() => setComposing(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
            <PrimaryButton onClick={submit} disabled={!text.trim()} progress={createStatus.isPending}>
              {t('send')}
            </PrimaryButton>
          </div>
        </Card>
      ) : (
        <button
          type="button"
          onClick={() => setComposing(true)}
          className="mt-2 flex w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint"
        >
          <Icon name="bolt" size={20} className="text-accent" />
          {t('pulseFree')}
        </button>
      )}

      <div className="flex flex-col gap-2 pt-3">
        {feed.isLoading ? <Skeleton className="h-24 w-full" /> : null}

        {feed.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            {t('retryConnection')}
          </Banner>
        ) : null}

        {feed.data && feed.data.length === 0 ? (
          <EmptyState icon="bolt" title={t('empty')} text={t('pulseEmpty')} />
        ) : null}

        {feed.data?.map((status) => {
          const left = minutesLeft(status.expires_at);
          return (
            <Card key={status.id} className="p-4">
              <div className="flex items-center gap-2">
                <Avatar name={status.author_name} size={36} />
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-body-strong text-ink">{status.author_name}</span>
                  <span className="text-caption text-ink-2">
                    {t('left')} {left} {t('minutesShort')}
                  </span>
                </div>
                <Chip tone={left > 15 ? 'success' : 'warning'} icon="schedule">
                  {left > 0 ? t('pulseFree15') : t('pulseExpired')}
                </Chip>
              </div>

              <p className="whitespace-pre-wrap pt-2 text-body-lg text-ink">{status.text}</p>

              <div className="flex items-center justify-between pt-3">
                <span className="flex items-center gap-1 text-body-secondary text-ink-2">
                  <Icon name="how_to_reg" size={18} />
                  {status.responses}
                </span>

                {status.mine ? (
                  <button
                    type="button"
                    onClick={() => closeStatus.mutate(status.id)}
                    className="text-body-secondary text-danger"
                  >
                    {t('pulseFinish')}
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={status.responded || left === 0}
                    onClick={() =>
                      respond.mutate(status.id, {
                        onSuccess: () => feedback(t('pulseRespondDone')),
                        onError: () => feedback(t('pulseClosed'), 'error'),
                      })
                    }
                    className={`h-9 rounded-xl px-4 text-body-secondary-strong ${
                      status.responded || left === 0
                        ? 'bg-neutral-soft text-ink-2'
                        : 'bg-accent-soft text-accent'
                    }`}
                  >
                    {status.responded ? t('pulseResponded') : t('pulseRespond')}
                  </button>
                )}
              </div>

              {status.mine && status.responses > 0 ? <ResponderList statusId={status.id} /> : null}
            </Card>
          );
        })}
      </div>

      <div className="px-1 pt-4 text-caption text-ink-2">{t('pulseNote')}</div>
    </Screen>
  );
}
