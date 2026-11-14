import { useMemo, useState } from 'react';
import { useDayEvents, useDayStatus, useWeather, type DayEvent } from '../api';
import { addDays, eventsOnDay, sortEvents, startOfWeek } from '../calendar';
import { useConnectDay, useDisconnectDay } from '../mutations';
import { EventComposer } from './EventComposer';
import { DayWeather, loadWeatherCity, WeatherPicker } from './WeatherPicker';
import { TimetableImporter } from './TimetableImporter';
import { EventEditor } from './EventEditor';
import { AppHeader, Banner, Card, EmptyState, Icon, PrimaryButton, Screen, SectionLabel, Segmented, Skeleton } from '@/shared/ui';
import { useT, useI18n } from '@/shared/i18n';
import { formatDayMonth, formatDateTime, formatWeekday } from '@/shared/i18n/date';
import { useSession } from '@/app/session';
import { useFeedback } from '@/shared/ui/toast';
import { showConfirm } from '@/shared/telegram/sdk';

function EventDetails({ event, onOpen }: { event: DayEvent; onOpen: () => void }) {
  const t = useT();
  const { lang } = useI18n();
  return (
    <Card><button type="button" onClick={onOpen} aria-label={`${t('dayEventDetails')}: ${event.title}`} className="flex min-h-16 w-full flex-col gap-1 rounded-[14px] p-4 text-left active:bg-neutral-soft">
      <span className="break-words text-body-strong text-ink [overflow-wrap:anywhere]">{event.title}</span>
      <span className="text-caption text-ink-2">{event.all_day ? t('dayAllDay') : `${formatDateTime(event.starts_at, lang)} - ${formatDateTime(event.ends_at, lang)}`}</span>
      {event.location ? <span className="break-words text-body-secondary text-ink-2 [overflow-wrap:anywhere]">{event.location}</span> : null}
    </button></Card>
  );
}

export function DayHomePage() {
  const t = useT();
  const { lang } = useI18n();
  const feedback = useFeedback();
  const { status: session } = useSession();
  const ready = session === 'ready';
  const status = useDayStatus(ready);
  const connect = useConnectDay();
  const disconnect = useDisconnectDay();
  const [view, setView] = useState<'week' | 'agenda'>('week');
  const [anchor, setAnchor] = useState(() => startOfWeek(new Date()));
  const [composing, setComposing] = useState(false);
  const [importing, setImporting] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [weatherCity, setWeatherCity] = useState(loadWeatherCity);
  const [confirmingDisconnect, setConfirmingDisconnect] = useState(false);
  const linked = status.data?.linked === true;
  const events = useDayEvents(anchor.toISOString(), addDays(anchor, 7).toISOString(), ready && linked);
  const weather = useWeather(weatherCity, Intl.DateTimeFormat().resolvedOptions().timeZone, ready);
  const agenda = useMemo(() => sortEvents(events.data ?? []), [events.data]);
  const days = useMemo(() => Array.from({ length: 7 }, (_, index) => {
    const date = addDays(anchor, index);
    return { date, events: eventsOnDay(agenda, date) };
  }), [anchor, agenda]);
  const busy = status.isFetching || events.isFetching || connect.isPending || disconnect.isPending;

  const refresh = () => { void status.refetch(); if (linked) void events.refetch(); if (weatherCity) void weather.refetch(); };
  const unlink = async () => {
    if (confirmingDisconnect || disconnect.isPending) return;
    setConfirmingDisconnect(true);
    try {
      if (await showConfirm(t('dayDisconnectConfirm'))) disconnect.mutate(undefined, {
        onSuccess: () => { setComposing(false); feedback(t('dayDisconnectDone')); },
        onError: () => feedback(t('saveFailed'), 'error'),
      });
    } finally { setConfirmingDisconnect(false); }
  };

  if (importing) return <Screen header={<AppHeader title={t('dayImport')} onBack={() => setImporting(false)} leadingIcon="document_scanner" />} bottomSpace><TimetableImporter week={anchor} available={status.data?.import_available === true} onClose={() => setImporting(false)} /></Screen>;
  if (editing) return <Screen header={<AppHeader title={t('dayEventDetails')} onBack={() => setEditing(null)} leadingIcon="event" />} bottomSpace><EventEditor id={editing} onClose={() => setEditing(null)} onSaved={(date) => { setEditing(null); setAnchor(startOfWeek(date)); }} /></Screen>;
  if (composing) return <Screen header={<AppHeader title={t('dayCreate')} onBack={() => setComposing(false)} leadingIcon="event" />} bottomSpace><div className="pt-3"><EventComposer onCancel={() => setComposing(false)} onCreated={(date) => { setComposing(false); setAnchor(startOfWeek(date)); }} /></div></Screen>;

  return (
    <Screen header={<AppHeader title="SruDay" leadingIcon="calendar_month" trailing={
      <button type="button" onClick={refresh} disabled={!ready || busy} aria-label={t('refresh')} className="flex size-11 items-center justify-center text-accent disabled:opacity-50"><Icon name="refresh" size={22} /></button>
    } />} bottomSpace>
      <div className="flex flex-col gap-3 pt-2">
        <p className="break-words text-caption text-ink-2">{t('dayDeviceTimezone')}: {Intl.DateTimeFormat().resolvedOptions().timeZone}</p>
        {ready ? <WeatherPicker city={weatherCity} onChange={setWeatherCity} /> : null}
        {weather.isError ? <Banner tone="warning" icon="cloud_off" title={t('dayWeatherUnavailable')} action={<button type="button" onClick={() => void weather.refetch()} className="min-h-11">{t('retry')}</button>} /> : null}
        {session === 'outside' ? <Banner tone="warning" icon="info" title={t('openInTelegram')} /> : null}
        {ready && status.isPending ? <Skeleton className="h-28 w-full" /> : null}
        {status.isError ? <Banner tone="danger" icon="error" title={t('dayUnavailable')} action={<button type="button" onClick={refresh} className="min-h-11 font-semibold">{t('retry')}</button>} /> : null}
        {status.data?.configured === false ? <Banner tone="info" icon="link_off" title={t('daySetupUnavailable')} /> : null}
        {status.data?.configured && !linked ? <>
          <Banner tone="info" icon="link" title={t('dayNotLinked')}>{t('dayNotLinkedText')}</Banner>
          <PrimaryButton full progress={connect.isPending} disabled={!ready} onClick={() => connect.mutate(undefined, {
            onSuccess: () => feedback(t('dayConnectDone')),
          })}>{t('dayConnect')}</PrimaryButton>
        </> : null}
        {connect.isError ? <Banner tone="danger" icon="error" title={t('dayConnectFailed')} /> : null}
        <>
          <div className="flex items-center justify-between gap-2">
            <button type="button" onClick={() => setAnchor((current) => addDays(current, -7))} aria-label={t('prevWeek')} className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-neutral-soft text-ink"><Icon name="chevron_left" size={22} /></button>
            <span className="text-center text-body-strong text-ink">{formatDayMonth(anchor, lang)} - {formatDayMonth(addDays(anchor, 6), lang)}</span>
            <button type="button" onClick={() => setAnchor((current) => addDays(current, 7))} aria-label={t('nextWeek')} className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-neutral-soft text-ink"><Icon name="chevron_right" size={22} /></button>
          </div>
          <button type="button" onClick={() => setAnchor(startOfWeek(new Date()))} className="min-h-11 self-center px-4 text-body-secondary text-accent">{t('dayToday')}</button>
          {linked ? <Segmented options={[{ value: 'week', label: t('dayWeek') }, { value: 'agenda', label: t('dayAgenda') }]} value={view} onChange={setView} /> : null}
          {linked ? <PrimaryButton full onClick={() => setComposing(true)}>{t('dayCreate')}</PrimaryButton> : null}
          {linked ? <button type="button" onClick={() => setImporting(true)} className="flex min-h-12 items-center justify-center gap-2 rounded-xl bg-surface px-4 text-body-strong text-accent"><Icon name="document_scanner" size={22} />{t('dayImport')}</button> : null}
          {events.isLoading ? <Skeleton className="h-28 w-full" /> : null}
          {events.isError ? <Banner tone="danger" icon="error" title={t('dayUnavailable')} action={<button type="button" onClick={() => void events.refetch()} className="min-h-11 font-semibold">{t('retry')}</button>} /> : null}
          {events.data?.length === 0 ? <EmptyState icon="event_available" title={t('dayNoEvents')} text={t('dayNoEventsText')} /> : null}
          {view === 'week' || !linked ? days.map((day) => <div key={day.date.toISOString()}>
            <SectionLabel>{formatWeekday(day.date, lang)}</SectionLabel>
            {weatherCity ? <DayWeather date={day.date} days={weather.data} loading={weather.isLoading} /> : null}
            <div className="flex flex-col gap-2">{linked && day.events.length ? day.events.map((event) => <EventDetails key={`${event.id}:${event.starts_at}`} event={event} onOpen={() => setEditing(event.id)} />) : <Card className="p-4 text-caption text-ink-2">{t(!linked ? 'dayConnectToSee' : events.isLoading ? 'loading' : events.isError ? 'dayUnavailable' : 'dayFree')}</Card>}</div>
          </div>) : agenda.map((event) => <EventDetails key={`${event.id}:${event.starts_at}`} event={event} onOpen={() => setEditing(event.id)} />)}
          {linked ? <button type="button" disabled={confirmingDisconnect || disconnect.isPending} onClick={() => void unlink()} className="min-h-11 px-4 text-body-secondary text-ink-2 disabled:opacity-50">{t('dayDisconnect')}</button> : null}
        </>
      </div>
    </Screen>
  );
}
