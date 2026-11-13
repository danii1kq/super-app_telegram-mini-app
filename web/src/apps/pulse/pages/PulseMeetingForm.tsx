import { useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import type { PulseStatus } from '../api';
import { useSavePulseMeeting } from '../mutations';
import { localDateTime, meetingInput } from '../time';
import { Banner, PrimaryButton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';
import { ApiError } from '@/shared/api/client';

export function PulseMeetingForm({ status, onClose }: { status: PulseStatus; onClose: () => void }) {
  const t = useT();
  const feedback = useFeedback();
  const save = useSavePulseMeeting();
  const submitting = useRef(false);
  const [title, setTitle] = useState(status.text.slice(0, 200));
  const [start, setStart] = useState(() => localDateTime(new Date()));
  const [end, setEnd] = useState(() => localDateTime(new Date(status.expires_at)));
  const input = meetingInput(title, start, end);
  const notLinked = save.error instanceof ApiError && save.error.code === 'not_linked';

  const submit = () => {
    if (!input || submitting.current) return;
    submitting.current = true;
    save.mutate(input, {
      onSuccess: () => { feedback(t('pulseCalendarSaved')); onClose(); },
      onSettled: () => { submitting.current = false; },
    });
  };

  return (
    <section aria-label={t('pulseCalendar')} className="mt-3 flex min-w-0 flex-col gap-3 border-t border-line pt-3">
      <p className="text-caption text-ink-2">{t('pulseCalendarHint')}</p>
      <label className="text-body-secondary text-ink">{t('pulseCalendarTitle')}
        <input value={title} onChange={(event) => setTitle(event.target.value)} maxLength={200} disabled={save.isPending}
          className="mt-1 h-11 w-full min-w-0 rounded-xl bg-neutral-soft/60 px-3 text-body-lg outline-none focus:ring-2 focus:ring-accent" />
      </label>
      <label className="min-w-0 text-body-secondary text-ink">{t('pulseCalendarStart')}
        <input type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)} disabled={save.isPending}
          className="mt-1 h-11 w-full min-w-0 max-w-full rounded-xl bg-neutral-soft/60 px-3 outline-none focus:ring-2 focus:ring-accent" />
      </label>
      <label className="min-w-0 text-body-secondary text-ink">{t('pulseCalendarEnd')}
        <input type="datetime-local" value={end} onChange={(event) => setEnd(event.target.value)} disabled={save.isPending}
          className="mt-1 h-11 w-full min-w-0 max-w-full rounded-xl bg-neutral-soft/60 px-3 outline-none focus:ring-2 focus:ring-accent" />
      </label>
      <p className="break-words text-caption text-ink-2">{t('pulseDeviceTimezone')}: {Intl.DateTimeFormat().resolvedOptions().timeZone}</p>
      {!input ? <p role="alert" className="text-caption text-danger">{t('pulseCalendarInvalid')}</p> : null}
      {save.isError ? <Banner tone="danger" icon="error" title={t(notLinked ? 'dayNotLinked' : 'pulseCalendarFailed')} /> : null}
      <Link to="/calendar" className="flex min-h-11 items-center text-body-secondary text-accent">{t('pulseOpenCalendar')}</Link>
      <div className="flex items-center justify-between gap-2">
        <button type="button" onClick={onClose} disabled={save.isPending} className="min-h-11 px-2 text-body-secondary text-ink-2 disabled:opacity-50">{t('cancel')}</button>
        <PrimaryButton onClick={submit} disabled={!input} progress={save.isPending}>{t('save')}</PrimaryButton>
      </div>
    </section>
  );
}
