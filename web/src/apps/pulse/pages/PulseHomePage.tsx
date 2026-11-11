import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { minutesLeft, usePulseFeed } from '../api';
import { useCloseStatus, useCreateStatus, useRespond } from '../mutations';
import {
  AppHeader,
  Avatar,
  Banner,
  Card,
  Chip,
  EmptyState,
  Icon,
  PrimaryButton,
  Screen,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { useT } from '@/shared/i18n';

const DURATIONS = [
  { value: 30, label: '30 мин' },
  { value: 60, label: '1 час' },
  { value: 120, label: '2 часа' },
] as const;

/** SruPulse: who is free right now. */
export function PulseHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const feed = usePulseFeed();
  const createStatus = useCreateStatus();
  const respond = useRespond();
  const closeStatus = useCloseStatus();

  const [composing, setComposing] = useState(false);
  const [text, setText] = useState('');
  const [minutes, setMinutes] = useState<number>(60);

  const submit = () => {
    const value = text.trim();
    if (!value || !me) return;
    const expiresAt = new Date(Date.now() + minutes * 60_000).toISOString();
    createStatus.mutate(
      { text: value, expiresAt, audience: [me.id] },
      {
        onSuccess: () => {
          setText('');
          setComposing(false);
          feedback('Статус опубликован');
        },
        onError: () => feedback('Не удалось опубликовать статус', 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title="SruPulse" onBack={() => navigate('/apps')} leadingIcon="bolt" />}>
      {composing ? (
        <Card className="mt-2 p-3">
          <textarea
            value={text}
            onChange={(event) => setText(event.target.value)}
            placeholder="Свободен до 16:20, хочу кофе"
            rows={2}
            maxLength={280}
            className="w-full resize-none rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <div className="flex items-center gap-1.5 pt-2">
            {DURATIONS.map((duration) => (
              <button
                key={duration.value}
                type="button"
                onClick={() => setMinutes(duration.value)}
                className={
                  minutes === duration.value
                    ? 'h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                    : 'h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                }
              >
                {duration.label}
              </button>
            ))}
          </div>
          <div className="flex items-center justify-between pt-2">
            <button type="button" onClick={() => setComposing(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
            <PrimaryButton onClick={submit} disabled={!text.trim()} progress={createStatus.isPending}>
              {t('send')}
            </PrimaryButton>
          </div>
        </Card>
      ) : (
        <button
          type="button"
          onClick={() => setComposing(true)}
          className="mt-2 flex w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint"
        >
          <Icon name="bolt" size={20} className="text-accent" />
          Я свободен сейчас
        </button>
      )}

      <div className="flex flex-col gap-2 pt-3">
        {feed.isLoading ? <Skeleton className="h-24 w-full" /> : null}

        {feed.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            Проверьте соединение и повторите.
          </Banner>
        ) : null}

        {feed.data && feed.data.length === 0 ? (
          <EmptyState
            icon="bolt"
            title={t('empty')}
            text="Статусы видны выбранной аудитории и истекают сами."
          />
        ) : null}

        {feed.data?.map((status) => {
          const left = minutesLeft(status.expires_at);
          return (
            <Card key={status.id} className="p-4">
              <div className="flex items-center gap-2">
                <Avatar name={status.author_name} size={36} />
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-body-strong text-ink">{status.author_name}</span>
                  <span className="text-caption text-ink-2">
                    ещё {left} мин
                  </span>
                </div>
                <Chip tone={left > 15 ? 'success' : 'warning'} icon="schedule">
                  {left > 0 ? 'свободен' : 'истёк'}
                </Chip>
              </div>

              <p className="whitespace-pre-wrap pt-2 text-body-lg text-ink">{status.text}</p>

              <div className="flex items-center justify-between pt-3">
                <span className="flex items-center gap-1 text-body-secondary text-ink-2">
                  <Icon name="how_to_reg" size={18} />
                  {status.responses}
                </span>

                {status.mine ? (
                  <button
                    type="button"
                    onClick={() => closeStatus.mutate(status.id)}
                    className="text-body-secondary text-danger"
                  >
                    Завершить
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={status.responded || left === 0}
                    onClick={() =>
                      respond.mutate(status.id, {
                        onSuccess: () => feedback('Отклик отправлен'),
                        onError: () => feedback('Статус уже завершён', 'error'),
                      })
                    }
                    className={`h-9 rounded-xl px-4 text-body-secondary-strong ${
                      status.responded || left === 0
                        ? 'bg-neutral-soft text-ink-2'
                        : 'bg-accent-soft text-accent'
                    }`}
                  >
                    {status.responded ? 'Вы откликнулись' : 'Откликнуться'}
                  </button>
                )}
              </div>
            </Card>
          );
        })}
      </div>

      <div className="px-1 pt-4 text-caption text-ink-2">
        Статус виден только выбранной аудитории и перестаёт принимать отклики после окончания времени.
        Календарь SruDay не раскрывает названия событий.
      </div>
    </Screen>
  );
}
