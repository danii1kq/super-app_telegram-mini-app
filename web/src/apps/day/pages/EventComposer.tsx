import { useRef, useState } from 'react';
import { localInput, parseLocalInput } from '../calendar';
import { useCreateDayEvent, useUpdateDayEvent } from '../mutations';
import type { DayEvent } from '../api';
import { ApiError } from '@/shared/api/client';
import { Banner, Card, PrimaryButton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';

export function EventComposer({ initial, onCreated, onCancel }: { initial?: DayEvent; onCreated: (start: Date) => void; onCancel: () => void }) {
  const t = useT();
  const feedback = useFeedback();
  const create = useCreateDayEvent();
  const update = useUpdateDayEvent();
  const original = useRef(initial);
  const sending = useRef(false);
  const [title, setTitle] = useState(initial?.title ?? '');
  const [location, setLocation] = useState(initial?.location ?? '');
  const [allDay, setAllDay] = useState(initial?.all_day ?? false);
  const [startValue, setStartValue] = useState(() => localInput(initial ? new Date(initial.starts_at) : new Date()));
  const [endValue, setEndValue] = useState(() => localInput(initial ? new Date(initial.ends_at) : new Date(Date.now() + 3_600_000)));
  const pending = create.isPending || update.isPending;
  const failed = original.current ? update.error : create.error;
  const start = parseLocalInput(startValue);
  const end = parseLocalInput(endValue);
  const validTimes = start && end && end.getTime() - start.getTime() >= 60_000 && end.getTime() - start.getTime() <= 30 * 86_400_000;
  const inputClass = 'h-12 min-w-0 w-full rounded-xl bg-neutral-soft/60 px-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent';

  const submit = () => {
    if (!title.trim() || !start || !end || !validTimes || sending.current) return;
    sending.current = true;
    const body = { title: title.trim(), starts_at: start.toISOString(), ends_at: end.toISOString(), location: location.trim(), all_day: allDay };
    const options = {
      onSuccess: () => { feedback(t(original.current ? 'dayEventUpdated' : 'dayEventCreated')); onCreated(start); },
      onSettled: () => { sending.current = false; },
    };
    if (original.current) update.mutate({ ...body, id: original.current.id, version: original.current.version ?? 0 }, options);
    else create.mutate(body, options);
  };

  return (
    <Card className="gap-3 p-3">
      <p className="text-caption text-ink-2">{t('dayPrivateHint')}</p>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayTitle')}
        <input value={title} onChange={(event) => setTitle(event.target.value)} maxLength={200} disabled={pending} className={inputClass} />
      </label>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayStarts')}
        <input type="datetime-local" value={startValue} onChange={(event) => setStartValue(event.target.value)} disabled={pending} className={inputClass} />
      </label>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayEnds')}
        <input type="datetime-local" value={endValue} onChange={(event) => setEndValue(event.target.value)} disabled={pending} className={inputClass} />
      </label>
      <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayLocation')}
        <input value={location} onChange={(event) => setLocation(event.target.value)} maxLength={200} disabled={pending} className={inputClass} />
      </label>
      <label className="flex min-h-11 items-center gap-2 text-body-secondary text-ink"><input type="checkbox" checked={allDay} onChange={(event) => setAllDay(event.target.checked)} disabled={pending} className="size-5 accent-accent" />{t('dayAllDay')}</label>
      {!validTimes ? <p role="alert" className="text-caption text-danger">{t('dayInvalidTimes')}</p> : null}
      {failed ? <Banner tone="danger" icon="error" title={t(failed instanceof ApiError && failed.code === 'day_event_changed' ? 'dayEventChanged' : original.current ? 'saveFailed' : 'dayCreateFailedCheck')} /> : null}
      <div className="flex flex-wrap justify-between gap-2">
        <button type="button" onClick={onCancel} disabled={pending} className="min-h-11 px-2 text-body-secondary text-ink-2 disabled:opacity-50">{t('cancel')}</button>
        <PrimaryButton onClick={submit} disabled={!title.trim() || !validTimes} progress={pending}>{t(original.current ? 'save' : 'dayCreate')}</PrimaryButton>
      </div>
    </Card>
  );
}
