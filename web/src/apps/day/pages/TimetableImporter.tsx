import { useEffect, useRef, useState } from 'react';
import { useCreateDayEvent, useReadTimetable, type TimetableItem } from '../mutations';
import { localDayKey, localInput, parseLocalInput } from '../calendar';
import { ApiError } from '@/shared/api/client';
import { useT } from '@/shared/i18n';
import { Banner, Card, Icon, PrimaryButton } from '@/shared/ui';

const inputClass = 'h-12 w-full min-w-0 rounded-xl bg-neutral-soft/60 px-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent disabled:opacity-60';

function TimetableRow({ item }: { item: TimetableItem }) {
  const t = useT();
  const create = useCreateDayEvent();
  const sending = useRef(false);
  const [title, setTitle] = useState(item.title);
  const [startValue, setStart] = useState(item.starts_at ? localInput(new Date(item.starts_at)) : '');
  const [endValue, setEnd] = useState(item.ends_at ? localInput(new Date(item.ends_at)) : '');
  const [location, setLocation] = useState(item.location);
  const [reviewed, setReviewed] = useState(false);
  const [state, setState] = useState<'draft' | 'saved' | 'unknown'>('draft');
  const start = parseLocalInput(startValue);
  const end = parseLocalInput(endValue);
  const valid = title.trim() && start && end && end.getTime() - start.getTime() >= 60_000 && end.getTime() - start.getTime() <= 30 * 86_400_000;
  const locked = create.isPending || state !== 'draft';
  const edit = (change: () => void) => { change(); setReviewed(false); };

  const save = () => {
    if (!reviewed || !valid || !start || !end || sending.current || state !== 'draft') return;
    sending.current = true;
    create.mutate({ title: title.trim(), starts_at: start.toISOString(), ends_at: end.toISOString(), location: location.trim() }, {
      onSuccess: () => setState('saved'),
      onError: (error) => {
        if (!(error instanceof ApiError && (error.status === 422 || error.code === 'not_linked'))) setState('unknown');
      },
      onSettled: () => { sending.current = false; },
    });
  };

  return <Card className="gap-3 p-3">
    {item.uncertain && state === 'draft' ? <Banner tone="warning" icon="help" title={t('dayImportUncertain')} /> : null}
    <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayTitle')}<input value={title} onChange={(event) => edit(() => setTitle(event.target.value))} maxLength={200} disabled={locked} className={inputClass} /></label>
    <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayStarts')}<input type="datetime-local" value={startValue} onChange={(event) => edit(() => setStart(event.target.value))} disabled={locked} className={inputClass} /></label>
    <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayEnds')}<input type="datetime-local" value={endValue} onChange={(event) => edit(() => setEnd(event.target.value))} disabled={locked} className={inputClass} /></label>
    <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayLocation')}<input value={location} onChange={(event) => edit(() => setLocation(event.target.value))} maxLength={200} disabled={locked} className={inputClass} /></label>
    {state === 'draft' ? <>
      {!valid ? <p role="alert" className="text-caption text-danger">{t('dayInvalidTimes')}</p> : null}
      {create.isError ? <Banner tone="danger" icon="error" title={t('dayCreateFailedCheck')} /> : null}
      <label className="flex min-h-11 items-start gap-2 py-2 text-caption text-ink"><input type="checkbox" checked={reviewed} onChange={(event) => setReviewed(event.target.checked)} disabled={locked || !valid} className="mt-0.5 size-5 shrink-0 accent-accent" /><span>{t('dayImportReviewed')}</span></label>
      <PrimaryButton full onClick={save} disabled={!reviewed || !valid} progress={create.isPending}>{t('dayCreate')}</PrimaryButton>
    </> : <Banner tone={state === 'saved' ? 'success' : 'warning'} icon={state === 'saved' ? 'check_circle' : 'help'} title={t(state === 'saved' ? 'dayImportSaved' : 'dayImportUnknown')} />}
  </Card>;
}

export function TimetableImporter({ week, available, onClose }: { week: Date; available: boolean; onClose: () => void }) {
  const t = useT();
  const read = useReadTimetable();
  const sending = useRef(false);
  const [weekValue, setWeek] = useState(() => localDayKey(week));
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState('');
  const [consent, setConsent] = useState(false);
  const [items, setItems] = useState<TimetableItem[] | null>(null);
  const [fileError, setFileError] = useState(false);
  const weekDate = parseLocalInput(`${weekValue}T00:00`);
  const validWeek = weekDate?.getDay() === 1;

  useEffect(() => {
    if (!file) { setPreview(''); return; }
    const url = URL.createObjectURL(file); setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  const recognize = () => {
    if (!file || !consent || !validWeek || !available || sending.current) return;
    sending.current = true;
    read.mutate({ file, week: weekValue, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone }, {
      onSuccess: (result) => setItems(result.items),
      onSettled: () => { sending.current = false; },
    });
  };

  if (items !== null) return <div className="flex flex-col gap-3 py-3">
    <p className="text-caption text-ink-2">{t('dayImportHint')}</p>
    {items.length === 0 ? <Banner tone="info" icon="event_busy" title={t('dayImportEmpty')} /> : items.map((item, index) => <TimetableRow key={index} item={item} />)}
    <button type="button" onClick={onClose} className="min-h-12 rounded-xl bg-surface px-3 text-body-strong text-accent"><Icon name="calendar_month" size={20} /> {t('dayBackCalendar')}</button>
  </div>;

  const rateLimited = read.error instanceof ApiError && read.error.code === 'import_rate_limit';
  return <Card className="mt-3 gap-3 p-3">
    <p className="text-caption text-ink-2">{t('dayImportHint')}</p>
    {!available ? <Banner tone="info" icon="info" title={t('dayImportNoKey')} /> : null}
    <label className="flex min-w-0 flex-col gap-1 text-caption text-ink-2">{t('dayImportWeek')}<input type="date" value={weekValue} onChange={(event) => setWeek(event.target.value)} disabled={read.isPending} className={inputClass} /></label>
    {!validWeek ? <p role="alert" className="text-caption text-danger">{t('dayImportWeek')}</p> : null}
    <label className="flex min-w-0 flex-col gap-2 text-caption text-ink-2">{t('dayImportPhoto')}<input type="file" accept="image/jpeg,image/png,image/webp" disabled={read.isPending || !available} onChange={(event) => {
      const next = event.target.files?.[0];
      const valid = next && next.size > 0 && next.size <= 10 * 1024 * 1024;
      setFileError(!!next && !valid); setFile(valid ? next : null); setConsent(false); event.target.value = '';
    }} className="min-h-12 min-w-0 max-w-full py-2 text-body-secondary text-ink" /></label>
    {fileError ? <p role="alert" className="text-caption text-danger">{t('dayImportLimit')}</p> : null}
    {preview ? <img src={preview} alt={t('dayImportPhoto')} className="max-h-60 w-full rounded-xl object-contain" /> : null}
    <label className="flex min-h-11 items-start gap-2 py-2 text-caption text-ink"><input type="checkbox" checked={consent} onChange={(event) => setConsent(event.target.checked)} disabled={read.isPending || !file} className="mt-0.5 size-5 shrink-0 accent-accent" /><span>{t('dayImportConsent')}</span></label>
    {read.isError ? <Banner tone="danger" icon="error" title={t(rateLimited ? 'dayImportRate' : 'dayImportFailed')} /> : null}
    <PrimaryButton full onClick={recognize} disabled={!file || !consent || !validWeek || !available} progress={read.isPending}>{t('dayImportRecognize')}</PrimaryButton>
  </Card>;
}
