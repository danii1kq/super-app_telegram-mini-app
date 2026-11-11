import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useDocOrder } from '../api';
import { newIdempotencyKey, useDocDeadline } from '@/shared/api/integrations';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  PrimaryButton,
  Row,
  Screen,
  SecondaryButton,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';

/** One SruDoc order, with the SruDoc -> SruDay deadline action. */
export function DocOrderPage() {
  const { number = '' } = useParams();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const order = useDocOrder(number);
  const deadline = useDocDeadline();

  const [picking, setPicking] = useState(false);
  const [dueDate, setDueDate] = useState('');
  // One key per attempt: a retry after a network error reuses it, so the
  // calendar does not get a second event.
  const [idempotencyKey, setIdempotencyKey] = useState(newIdempotencyKey);

  const createDeadline = () => {
    if (!dueDate) return;
    deadline.mutate(
      {
        idempotencyKey,
        orderNumber: number,
        title: order.data?.title ? `Дедлайн: ${order.data.title}` : '',
        dueDate: new Date(`${dueDate}T09:00:00`).toISOString(),
      },
      {
        onSuccess: () => {
          setPicking(false);
          setIdempotencyKey(newIdempotencyKey());
          feedback('Дедлайн добавлен в SruDay');
        },
        onError: () => feedback('Не удалось создать дедлайн', 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title={`Заказ ${number}`} onBack={() => navigate('/apps/doc')} />}>
      {order.isLoading ? <Skeleton className="mt-2 h-24 w-full" /> : null}

      {order.isError ? (
        <div className="pt-2">
          <Banner tone="danger" icon="error" title="Заказ недоступен">
            Проверьте привязку аккаунта SruDoc.
          </Banner>
        </div>
      ) : null}

      {order.data ? (
        <div className="pt-2">
          <Card>
            <Row title={order.data.title || 'Без названия'} subtitle={`№ ${order.data.number}`} />
            <Row title="Статус" value={<Chip tone="accent">{order.data.status}</Chip>} />
            <Row title="Сумма" value={`${order.data.amount_minor} ₸`} />
            <Row
              title="Результат"
              subtitle={order.data.has_result ? 'Файл доступен' : 'Ещё не готов'}
              icon={order.data.has_result ? 'download' : 'schedule'}
              iconTone={order.data.has_result ? 'success' : 'ink-2'}
            />
          </Card>

          {picking ? (
            <Card className="mt-3 p-3">
              <label className="text-caption text-ink-2" htmlFor="due-date">
                Дата сдачи
              </label>
              <input
                id="due-date"
                type="date"
                value={dueDate}
                onChange={(event) => setDueDate(event.target.value)}
                className="mt-1 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none"
              />
              <div className="flex items-center justify-between pt-2">
                <button
                  type="button"
                  onClick={() => setPicking(false)}
                  className="text-body-secondary text-ink-2"
                >
                  Отмена
                </button>
                <PrimaryButton onClick={createDeadline} disabled={!dueDate} progress={deadline.isPending}>
                  Создать событие
                </PrimaryButton>
              </div>
            </Card>
          ) : (
            <div className="pt-3">
              <SecondaryButton onClick={() => setPicking(true)} full>
                Добавить дедлайн в SruDay
              </SecondaryButton>
            </div>
          )}

          <div className="px-1 pt-3 text-caption text-ink-2">
            Файл и оплата остаются в SruDoc: SruNal показывает статус, но не хранит документ. Дедлайн
            создаётся личной копией события со ссылкой на заказ.
          </div>
        </div>
      ) : null}
    </Screen>
  );
}
