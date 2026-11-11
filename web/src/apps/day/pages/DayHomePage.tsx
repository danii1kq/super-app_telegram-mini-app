import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useDayEvents, useDayStatus } from '../api';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  EmptyState,
  Hairline,
  Icon,
  Row,
  Screen,
  SectionLabel,
  Segmented,
  Skeleton,
} from '@/shared/ui';
import { useT, useI18n } from '@/shared/i18n';
import { formatMonthDay, formatTime, formatWeekday, formatAgenda } from '@/shared/i18n/date';
import { useSession } from '@/app/session';

function startOfWeek(date: Date): Date {
  const copy = new Date(date);
  const day = (copy.getDay() + 6) % 7; // Monday first
  copy.setDate(copy.getDate() - day);
  copy.setHours(0, 0, 0, 0);
  return copy;
}

function formatDay(date: Date, lang: 'ru' | 'kk' | 'en'): string {
  return formatWeekday(date, lang);
}

/** SruDay space: the caller's calendar week from the existing SruDay service. */
export function DayHomePage() {
  const t = useT();
  const { lang } = useI18n();
  const navigate = useNavigate();
  const { status: session } = useSession();
  const status = useDayStatus(session === 'ready');
  const [view, setView] = useState<'week' | 'agenda'>('week');
  const [anchor, setAnchor] = useState(() => startOfWeek(new Date()));

  const from = anchor.toISOString();
  const events = useDayEvents(from, status.data === true);

  const days = useMemo(() => {
    const out: { date: Date; events: typeof events.data }[] = [];
    for (let index = 0; index < 7; index += 1) {
      const date = new Date(anchor);
      date.setDate(anchor.getDate() + index);
      const dayEvents = (events.data ?? []).filter((event) => {
        const start = new Date(event.starts_at);
        return start.toDateString() === date.toDateString();
      });
      out.push({ date, events: dayEvents });
    }
    return out;
  }, [anchor, events.data]);

  const shift = (weeks: number) => {
    const next = new Date(anchor);
    next.setDate(anchor.getDate() + weeks * 7);
    setAnchor(next);
  };

  return (
    <Screen header={<AppHeader title="SruDay" onBack={() => navigate('/apps')} leadingIcon="calendar_month" />}>
      {status.data === false ? (
        <div className="pt-2">
          <Banner tone="info" icon="link" title={t('dayNotLinked')}>
            {t('dayNotLinkedText')}
          </Banner>
        </div>
      ) : null}

      {status.data === true ? (
        <>
          <div className="flex items-center justify-between pt-2">
            <button
              type="button"
              onClick={() => shift(-1)}
              aria-label={t('prevWeek')}
              className="flex size-10 items-center justify-center rounded-xl bg-neutral-soft text-ink"
            >
              <Icon name="chevron_left" size={22} />
            </button>
            <span className="text-body-strong text-ink">{formatMonthDay(anchor, lang)}</span>
            <button
              type="button"
              onClick={() => shift(1)}
              aria-label={t('nextWeek')}
              className="flex size-10 items-center justify-center rounded-xl bg-neutral-soft text-ink"
            >
              <Icon name="chevron_right" size={22} />
            </button>
          </div>

          <div className="pt-2">
            <Segmented
              options={[
                { value: 'week', label: t('dayWeek') },
                { value: 'agenda', label: t('dayAgenda') },
              ]}
              value={view}
              onChange={setView}
            />
          </div>

          {events.isLoading ? <Skeleton className="mt-3 h-24 w-full" /> : null}

          {events.isError ? (
            <div className="pt-3">
              <Banner tone="danger" icon="error" title={t('error')}>
                {t('dayUnavailable')}
              </Banner>
            </div>
          ) : null}

          {events.data && events.data.length === 0 ? (
            <EmptyState icon="event_available" title={t('dayNoEvents')} text={t('dayNoEventsText')} />
          ) : null}

          {view === 'week'
            ? days.map((day) => (
                <div key={day.date.toISOString()}>
                  <SectionLabel>{formatDay(day.date, lang)}</SectionLabel>
                  <Card>
                    {day.events && day.events.length > 0 ? (
                      day.events.map((event, index) => (
                        <div key={event.id}>
                          {index > 0 ? <Hairline /> : null}
                          <Row
                            icon="event"
                            title={event.title}
                            subtitle={
                              event.all_day
                                ? t('dayAllDay')
                                : `${formatTime(event.starts_at, lang)} - ${formatTime(event.ends_at, lang)}`
                            }
                            trailing={event.source_ref ? <Chip tone="accent">SruDoc</Chip> : undefined}
                          />
                        </div>
                      ))
                    ) : (
                      <div className="px-4 py-3 text-caption text-ink-2">{t('dayFree')}</div>
                    )}
                  </Card>
                </div>
              ))
            : events.data?.map((event) => (
                <div key={event.id} className="pt-2">
                  <Card>
                    <Row
                      icon="event"
                      title={event.title}
                      subtitle={formatAgenda(event.starts_at, lang)}
                      trailing={event.source_ref ? <Chip tone="accent">SruDoc</Chip> : undefined}
                    />
                  </Card>
                </div>
              ))}
        </>
      ) : null}
    </Screen>
  );
}
