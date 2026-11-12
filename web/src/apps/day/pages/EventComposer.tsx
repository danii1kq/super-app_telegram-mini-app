import { useRef, useState } from 'react';
import { localInput, parseLocalInput } from '../calendar';
import { useCreateDayEvent } from '../mutations';
import { Banner, Card, PrimaryButton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';

export function EventComposer({ onCreated, onCancel }: { onCreated: (start: Date) => void; onCancel: () => void }) {
  const t = useT();
  const feedback = useFeedback();
  const create = useCreateDayEvent();
  const sending = useRef(false);
  const [title, setTitle] = useState('');
  const [location, setLocation] = useState('');
  const [startValue, setStartValue] = useState(() => localInput(new Date()));
  const [endValue, setEndValue] = useState(() => localInput(new Date(Date.now() + 3_600_000)));
  const start = parseLocalInput(startValue);
  const end = parseLocalInput(endValue);
  const validTimes = start && end && end.getTime() - start.getTime() >= 60_000 && end.getTime() - start.getTime() <= 30 * 86_400_000;
  const inputClass = 'h-12 min-w-0 w-full rounded-xl bg-neutral-soft/60 px-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent';

  const submit = () => {
    if (!title.trim() || !start || !end || !validTimes || sending.current) return;
    sending.current = true;
    create.mutate({ title: title.trim(), starts_at: start.toISOString(), ends_at: end.toISOString(), location: location.trim() }, {
      onSuccess: () => { feedback(t('dayEventCreated')); onCreated(start); },
      onSettled: () => { sending.current = false; },
    });
  };

  return (
    <Card className="gap-3 p-3">
      <p className="text-caption text-ink-2">{t('dayPrivateHint')}</p>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayTitle')}
        <input value={title} onChange={(event) => setTitle(event.target.value)} maxLength={200} disabled={create.isPending} className={inputClass} />
      </label>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayStarts')}
        <input type="datetime-local" value={startValue} onChange={(event) => setStartValue(event.target.value)} disabled={create.isPending} className={inputClass} />
      </label>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayEnds')}
        <input type="datetime-local" value={endValue} onChange={(event) => setEndValue(event.target.value)} disabled={create.isPending} className={inputClass} />
      </label>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayLocation')}
        <input value={location} onChange={(event) => setLocation(event.target.value)} maxLength={200} disabled={create.isPending} className={inputClass} />
      </label>
      {!validTimes ? <p role="alert" className="text-caption text-danger">{t('dayInvalidTimes')}</p> : null}
      {create.isError ? <Banner tone="danger" icon="error" title={t('dayCreateFailedCheck')} /> : null}
      <div className="flex flex-wrap justify-between gap-2">
        <button type="button" onClick={onCancel} disabled={create.isPending} className="min-h-11 px-2 text-body-secondary text-ink-2 disabled:opacity-50">{t('cancel')}</button>
        <PrimaryButton onClick={submit} disabled={!title.trim() || !validTimes} progress={create.isPending}>{t('dayCreate')}</PrimaryButton>
      </div>
    </Card>
  );
}
