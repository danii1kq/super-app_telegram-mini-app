import { useDayEvent } from '../api';
import { EventComposer } from './EventComposer';
import { Banner, Card, Skeleton } from '@/shared/ui';
import { useI18n, useT } from '@/shared/i18n';
import { formatDateTime } from '@/shared/i18n/date';

export function EventEditor({ id, onClose, onSaved }: { id: string; onClose: () => void; onSaved: (date: Date) => void }) {
  const t = useT();
  const { lang } = useI18n();
  const query = useDayEvent(id);
  const event = query.data;
  return <div className="flex flex-col gap-3 py-3">
    {query.isLoading ? <Skeleton className="h-40 w-full" /> : null}
    {query.isError ? <Banner tone="danger" icon="error" title={t('dayUnavailable')} action={<button type="button" onClick={() => void query.refetch()} className="min-h-11">{t('retry')}</button>} /> : null}
    {event ? event.editable ? <EventComposer key={event.id} initial={event} onCreated={onSaved} onCancel={onClose} /> : <>
      <Card className="gap-2 p-4"><span className="break-words text-body-strong text-ink">{event.title}</span><span className="text-caption text-ink-2">{formatDateTime(event.starts_at, lang)} - {formatDateTime(event.ends_at, lang)}</span>{event.location ? <span className="break-words text-body-secondary text-ink-2">{event.location}</span> : null}</Card>
      <Banner tone="info" icon="info" title={t('dayEventReadonly')} />
    </> : null}
  </div>;
}
