import { useNavigate } from 'react-router-dom';
import { useDocOrders, useDocStatus } from '../api';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  EmptyState,
  Hairline,
  Row,
  Screen,
  Skeleton,
} from '@/shared/ui';
import { useT } from '@/shared/i18n';

const STATUS_LABEL: Record<string, string> = {
  draft: 'Черновик',
  awaiting_payment: 'Ожидает оплаты',
  paid: 'Оплачен',
  generating: 'Готовится',
  ready: 'Готов',
  revision: 'Доработка',
  cancelled: 'Отменён',
};

function statusTone(status: string): 'accent' | 'success' | 'warning' | 'danger' | 'neutral' {
  if (status === 'ready') return 'success';
  if (status === 'cancelled') return 'danger';
  if (status === 'awaiting_payment' || status === 'revision') return 'warning';
  if (status === 'generating' || status === 'paid') return 'accent';
  return 'neutral';
}

/** SruDoc space: the caller's orders from the existing DocGen service. */
export function DocHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const status = useDocStatus();
  const orders = useDocOrders(status.data === true);

  return (
    <Screen header={<AppHeader title="SruDoc" onBack={() => navigate('/apps')} leadingIcon="description" />}>
      {status.data === false ? (
        <div className="pt-2">
          <Banner tone="info" icon="link" title="Аккаунт SruDoc не привязан">
            Заказы и файлы остаются в SruDoc. Привязка откроет их здесь и позволит связать документ с
            дедлайном в SruDay.
          </Banner>
        </div>
      ) : null}

      {status.data === true ? (
        <>
          <div className="pt-2">
            <Row
              icon="add_circle"
              title="Новый заказ"
              subtitle="Откроется сценарий SruDoc"
              onClick={() => navigate('/apps/doc/new')}
            />
          </div>

          <div className="flex flex-col gap-2 pt-3">
            {orders.isLoading ? <Skeleton className="h-20 w-full" /> : null}

            {orders.isError ? (
              <Banner tone="danger" icon="error" title={t('error')}>
                SruDoc сейчас недоступен. Остальные разделы SruNal работают.
              </Banner>
            ) : null}

            {orders.data && orders.data.length === 0 ? (
              <EmptyState icon="description" title="Заказов пока нет" text="Создайте первый заказ в SruDoc." />
            ) : null}

            {orders.data && orders.data.length > 0 ? (
              <Card>
                {orders.data.map((order, index) => (
                  <div key={order.number}>
                    {index > 0 ? <Hairline /> : null}
                    <Row
                      icon="description"
                      title={order.title || `Заказ ${order.number}`}
                      subtitle={`№ ${order.number}`}
                      onClick={() => navigate(`/apps/doc/order/${order.number}`)}
                      trailing={<Chip tone={statusTone(order.status)}>{STATUS_LABEL[order.status] ?? order.status}</Chip>}
                    />
                  </div>
                ))}
              </Card>
            ) : null}
          </div>
        </>
      ) : null}
    </Screen>
  );
}
