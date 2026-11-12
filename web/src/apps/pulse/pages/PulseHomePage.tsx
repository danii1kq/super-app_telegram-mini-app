import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { type PulseStatus, usePulseFeed } from '../api';
import { useCreateStatus } from '../mutations';
import { PulseCard } from './PulseCard';
import { PulseMeetingForm } from './PulseMeetingForm';
import { AppHeader, Avatar, Banner, Card, EmptyState, Icon, PrimaryButton, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { type Person, usePeople } from '@/core/social/api';
import { useT } from '@/shared/i18n';

const DURATIONS = [30, 60, 120] as const;

export function PulseHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { status } = useSession();
  const ready = status === 'ready';
  const feed = usePulseFeed(ready);
  const createStatus = useCreateStatus();
  const [composing, setComposing] = useState(false);
  const [text, setText] = useState('');
  const [minutes, setMinutes] = useState<number>(60);
  const [audience, setAudience] = useState<Person[]>([]);
  const [search, setSearch] = useState('');
  const [now, setNow] = useState(Date.now);
  const [meeting, setMeeting] = useState<PulseStatus | null>(null);
  const submitting = useRef(false);
  const people = usePeople(ready && composing, search.trim());
  const active = feed.data?.filter((item) => !item.closed_at && Date.parse(item.expires_at) > now);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const toggleAudience = (person: Person) => {
    setAudience((current) => current.some((item) => item.id === person.id)
      ? current.filter((item) => item.id !== person.id) : current.length < 100 ? [...current, person] : current);
  };

  const submit = () => {
    const value = text.trim();
    if (!value || !ready || audience.length === 0 || submitting.current) return;
    submitting.current = true;
    createStatus.mutate({
      text: value,
      expiresAt: new Date(Date.now() + minutes * 60_000).toISOString(),
      audience: audience.map((person) => person.id),
    }, {
      onSuccess: () => { setText(''); setAudience([]); setComposing(false); setSearch(''); feedback(t('pulsePublished')); },
      onError: () => feedback(t('pulseFailed'), 'error'),
      onSettled: () => { submitting.current = false; },
    });
  };

  if (meeting) {
    return <Screen header={<AppHeader title="SruDay" onBack={() => setMeeting(null)} leadingIcon="event" />}>
      <Card className="mt-2 p-4"><PulseMeetingForm key={meeting.id} status={meeting} onClose={() => setMeeting(null)} /></Card>
    </Screen>;
  }

  return (
    <Screen header={<AppHeader title="SruPulse" onBack={() => navigate('/apps')} leadingIcon="bolt" trailing={
      <button type="button" aria-label={t('refresh')} disabled={!ready || feed.isFetching} onClick={() => void feed.refetch()}
        className="flex size-11 items-center justify-center text-accent disabled:opacity-50"><Icon name="refresh" size={22} /></button>
    } />}>
      {status === 'outside' ? <div className="pt-2"><Banner tone="warning" icon="info" title={t('openInTelegram')} /></div> : null}
      {composing ? (
        <Card className="mt-2 gap-3 p-3">
          <textarea value={text} onChange={(event) => setText(event.target.value)} placeholder={t('pulsePlaceholder')}
            aria-label={t('pulsePlaceholder')} rows={3} maxLength={280} disabled={createStatus.isPending}
            className="w-full resize-y rounded-xl bg-neutral-soft/60 p-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent placeholder:text-hint disabled:opacity-50" />
          <div className="flex flex-wrap gap-2">
            {DURATIONS.map((duration) => <button key={duration} type="button" aria-pressed={minutes === duration} disabled={createStatus.isPending}
              onClick={() => setMinutes(duration)} className={`min-h-11 rounded-full px-3 text-caption ${minutes === duration ? 'bg-accent-soft font-semibold text-accent' : 'bg-neutral-soft text-ink-2'}`}>
              {duration} {t('minutesShort')}
            </button>)}
          </div>
          <p className="text-caption text-ink-2">{t('pulseAudienceHint')}</p>
          <span className="text-body-secondary-strong text-ink">{t('pulseRecipientsSelected')}: {audience.length}/100</span>
          {audience.length ? <div className="flex flex-wrap gap-2">{audience.map((person) =>
            <button key={person.id} type="button" aria-label={`${t('delete')}: ${person.public_name}`} disabled={createStatus.isPending}
              onClick={() => toggleAudience(person)} className="flex min-h-11 max-w-full items-center gap-1 rounded-full bg-accent-soft px-3 text-caption text-accent">
              <span className="min-w-0 break-words">{person.public_name}</span><Icon name="close" size={16} />
            </button>)}</div> : <p className="text-caption text-ink-2">{t('pulseChooseAudience')}</p>}
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder={t('peopleSearch')} aria-label={t('peopleSearch')}
            disabled={createStatus.isPending} className="h-11 w-full min-w-0 rounded-xl bg-neutral-soft/60 px-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent placeholder:text-hint" />
          {people.isLoading ? <Skeleton className="h-14 w-full" /> : null}
          {people.isError ? <Banner tone="danger" icon="error" title={t('error')} action={<button type="button" onClick={() => void people.refetch()} className="min-h-11">{t('retry')}</button>} /> : null}
          <div className="max-h-60 overflow-y-auto">
            {people.data?.map((person) => {
              const picked = audience.some((item) => item.id === person.id);
              return <button key={person.id} type="button" role="checkbox" aria-checked={picked} disabled={createStatus.isPending || (!picked && audience.length >= 100)}
                onClick={() => toggleAudience(person)} className="flex min-h-14 w-full items-center gap-2 px-1 text-left disabled:opacity-50">
                <Avatar name={person.public_name} size={32} /><span className="min-w-0 flex-1 break-words text-body-secondary text-ink">{person.public_name}</span>
                <Icon name={picked ? 'check_box' : 'check_box_outline_blank'} size={22} className="shrink-0 text-accent" />
              </button>;
            })}
          </div>
          {people.data?.length === 0 ? <p className="text-caption text-ink-2">{t('peopleEmpty')}</p> : null}
          <div className="flex items-center justify-between gap-2">
            <button type="button" disabled={createStatus.isPending} onClick={() => setComposing(false)} className="min-h-11 px-2 text-body-secondary text-ink-2 disabled:opacity-50">{t('cancel')}</button>
            <span className="text-caption tabular-nums text-ink-2">{text.length}/280</span>
            <PrimaryButton onClick={submit} disabled={!text.trim() || !audience.length || !ready} progress={createStatus.isPending}>{t('send')}</PrimaryButton>
          </div>
        </Card>
      ) : (
        <button type="button" disabled={!ready} onClick={() => setComposing(true)} className="mt-2 flex min-h-14 w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint disabled:opacity-50">
          <Icon name="bolt" size={22} className="text-accent" />{t('pulseFree')}
        </button>
      )}
      <div className="flex flex-col gap-3 pt-3">
        {feed.isLoading ? <Skeleton className="h-28 w-full" /> : null}
        {feed.isError ? <Banner tone="danger" icon="error" title={t('error')} action={<button type="button" onClick={() => void feed.refetch()} className="min-h-11">{t('retry')}</button>}>{t('retryConnection')}</Banner> : null}
        {ready && active?.length === 0 ? <EmptyState icon="bolt" title={t('empty')} text={t('pulseEmpty')} /> : null}
        {active?.map((item) => <PulseCard key={item.id} status={item} now={now} onCalendar={setMeeting} />)}
      </div>
      <p className="px-1 pt-4 text-caption text-ink-2">{t('pulseNote')}</p>
    </Screen>
  );
}
