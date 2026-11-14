import { useState } from 'react';
import { Link } from 'react-router-dom';
import { type PulseStatus, usePulseResponders } from '../api';
import { useCancelResponse, useCloseStatus, useRespond } from '../mutations';
import { minutesLeft } from '../time';
import { Avatar, Banner, Card, Chip, Icon, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';
import { showConfirm } from '@/shared/telegram/sdk';

export function PulseCard({ status, now, onCalendar }: { status: PulseStatus; now: number; onCalendar: (status: PulseStatus) => void }) {
  const t = useT();
  const feedback = useFeedback();
  const respond = useRespond();
  const withdraw = useCancelResponse();
  const close = useCloseStatus();
  const [showPeople, setShowPeople] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const people = usePulseResponders(status.id, status.mine && showPeople);
  const left = minutesLeft(status.expires_at, now);
  const busy = respond.isPending || withdraw.isPending;

  const finish = async () => {
    if (confirming || close.isPending) return;
    setConfirming(true);
    try {
      if (await showConfirm(t('pulseConfirmClose'))) {
        close.mutate(status.id, { onError: () => feedback(t('pulseActionFailed'), 'error') });
      }
    } finally { setConfirming(false); }
  };

  return (
    <Card className="p-4">
      <div className="flex items-center gap-2">
        <Avatar name={status.author_name} size={36} />
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-body-strong text-ink">{status.author_name}</span>
          <span className="text-caption text-ink-2">{t('left')} {left} {t('minutesShort')}</span>
        </div>
        <Chip tone={left > 15 ? 'success' : 'warning'} icon="schedule">{t(left > 0 ? 'pulseFree15' : 'pulseExpired')}</Chip>
      </div>
      <p className="whitespace-pre-wrap break-words pt-3 text-body-lg text-ink [overflow-wrap:anywhere]">{status.text}</p>
      <div className="flex flex-wrap items-center justify-between gap-2 pt-2">
        {status.mine ? (
          <button type="button" onClick={() => setShowPeople((value) => !value)} aria-expanded={showPeople}
            className="flex min-h-11 items-center gap-1 text-body-secondary text-accent"><Icon name="how_to_reg" size={18} />{t('pulseResponses')} · {status.responses}</button>
        ) : <span className="flex items-center gap-1 text-body-secondary text-ink-2"><Icon name="how_to_reg" size={18} />{status.responses}</span>}
        {status.mine ? (
          <button type="button" disabled={confirming || close.isPending} onClick={() => void finish()} className="min-h-11 px-2 text-body-secondary text-danger disabled:opacity-50">{t('pulseFinish')}</button>
        ) : (
          <button type="button" disabled={busy || left === 0} onClick={() => {
            const mutation = status.responded ? withdraw : respond;
            mutation.mutate(status.id, {
              onSuccess: () => feedback(t(status.responded ? 'pulseWithdrawn' : 'pulseRespondDone')),
              onError: () => feedback(t('pulseActionFailed'), 'error'),
            });
          }} className="min-h-11 rounded-xl bg-accent-soft px-4 text-body-secondary-strong text-accent disabled:opacity-50">
            {t(busy ? 'loading' : status.responded ? 'pulseWithdraw' : 'pulseRespond')}
          </button>
        )}
      </div>
      <div className="flex flex-wrap gap-2 border-t border-line pt-2">
        {!status.mine ? <Link to={`/people/${status.author_id}`} className="flex min-h-11 items-center gap-1 px-2 text-body-secondary text-accent"><Icon name="person" size={18} />{t('profile')}</Link> : null}
        <button type="button" onClick={() => onCalendar(status)} className="flex min-h-11 items-center gap-1 px-2 text-body-secondary text-accent"><Icon name="event" size={18} />{t('pulseCalendar')}</button>
      </div>
      {status.mine && showPeople ? (
        <section aria-label={t('pulseResponses')} className="border-t border-line pt-3">
          <p className="pb-2 text-caption text-ink-2">{t('pulseResponsesPrivate')}</p>
          {people.isPending ? <Skeleton className="h-11 w-full" /> : null}
          {people.isError ? <Banner tone="danger" icon="error" title={t('error')} action={<button type="button" onClick={() => void people.refetch()} className="min-h-11">{t('retry')}</button>} /> : null}
          {people.data?.length === 0 ? <p className="text-caption text-ink-2">{t('pulseNoResponses')}</p> : null}
          {people.data?.map((person) => (
            <div key={person.id} className="flex min-h-14 items-center gap-2">
              <Avatar name={person.public_name} size={32} />
              <span className="min-w-0 flex-1 break-words text-body-secondary text-ink">{person.public_name}</span>
              <Link to={`/people/${person.id}`} aria-label={`${t('profile')}: ${person.public_name}`} className="flex size-11 shrink-0 items-center justify-center text-accent"><Icon name="person" size={20} /></Link>
            </div>
          ))}
        </section>
      ) : null}
    </Card>
  );
}
