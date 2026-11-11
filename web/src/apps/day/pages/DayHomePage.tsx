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
import { useT } from '@/shared/i18n';

function startOfWeek(date: Date): Date {
  const copy = new Date(date);
  const day = (copy.getDay() + 6) % 7; // Monday first
  copy.setDate(copy.getDate() - day);
  copy.setHours(0, 0, 0, 0);
  return copy;
}

function formatDay(date: Date): string {
  return date.toLocaleDateString('ru-RU', { weekday: 'short', day: 'numeric', month: 'short' });
}

/** SruDay space: the caller's calendar week from the existing SruDay service. */
export function DayHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const status = useDayStatus();
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
          <Banner tone="info" icon="link" title="Аккаунт SruDay не привязан">
            Календарь, друзья и группы остаются в SruDay. Привязка откроет расписание здесь и позволит
            создавать дедлайны из SruDoc.
          </Banner>
        </div>
      ) : null}

      {status.data === true ? (
        <>
          <div className="flex items-center justify-between pt-2">
            <button
              type="button"
              onClick={() => shift(-1)}
              aria-label="Предыдущая неделя"
              className="flex size-10 items-center justify-center rounded-xl bg-neutral-soft text-ink"
            >
              <Icon name="chevron_left" size={22} />
            </button>
            <span className="text-body-strong text-ink">
              {anchor.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })}
            </span>
            <button
              type="button"
              onClick={() => shift(1)}
              aria-label="Следующая неделя"
              className="flex size-10 items-center justify-center rounded-xl bg-neutral-soft text-ink"
            >
              <Icon name="chevron_right" size={22} />
            </button>
          </div>

          <div className="pt-2">
            <Segmented
              options={[
                { value: 'week', label: 'Неделя' },
                { value: 'agenda', label: 'Список' },
              ]}
              value={view}
              onChange={setView}
            />
          </div>

          {events.isLoading ? <Skeleton className="mt-3 h-24 w-full" /> : null}

          {events.isError ? (
            <div className="pt-3">
              <Banner tone="danger" icon="error" title={t('error')}>
                SruDay сейчас недоступен. Остальные разделы SruNal работают.
              </Banner>
            </div>
          ) : null}

          {events.data && events.data.length === 0 ? (
            <EmptyState icon="event_available" title="Событий нет" text="На этой неделе календарь свободен." />
          ) : null}

          {view === 'week'
            ? days.map((day) => (
                <div key={day.date.toISOString()}>
                  <SectionLabel>{formatDay(day.date)}</SectionLabel>
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
                                ? 'Весь день'
                                : `${new Date(event.starts_at).toLocaleTimeString('ru-RU', {
                                    hour: '2-digit',
                                    minute: '2-digit',
                                  })} - ${new Date(event.ends_at).toLocaleTimeString('ru-RU', {
                                    hour: '2-digit',
                                    minute: '2-digit',
                                  })}`
                            }
                            trailing={
                              event.source_ref ? <Chip tone="accent">SruDoc</Chip> : undefined
                            }
                          />
                        </div>
                      ))
                    ) : (
                      <div className="px-4 py-3 text-caption text-ink-2">Свободно</div>
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
                      subtitle={new Date(event.starts_at).toLocaleString('ru-RU', {
                        weekday: 'short',
                        day: 'numeric',
                        month: 'short',
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
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
