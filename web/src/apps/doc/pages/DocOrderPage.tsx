import { useNavigate, useParams } from 'react-router-dom';
import { useDocOrder } from '../api';
import { AppHeader, Banner, Card, Chip, Row, Screen, Skeleton } from '@/shared/ui';

/** One SruDoc order. */
export function DocOrderPage() {
  const { number = '' } = useParams();
  const navigate = useNavigate();
  const order = useDocOrder(number);

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
          <div className="px-1 pt-3 text-caption text-ink-2">
            Файл и оплата остаются в SruDoc: SruNal показывает статус, но не хранит документ.
          </div>
        </div>
      ) : null}
    </Screen>
  );
}
