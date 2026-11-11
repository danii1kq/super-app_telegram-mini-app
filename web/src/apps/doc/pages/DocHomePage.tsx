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
import { useT, type DictKey } from '@/shared/i18n';
import { useSession } from '@/app/session';

const STATUS_KEY: Record<string, DictKey> = {
  draft: 'statusDraft',
  awaiting_payment: 'statusAwaitingPayment',
  paid: 'statusPaid',
  generating: 'statusGenerating',
  ready: 'statusReady',
  revision: 'statusRevision',
  cancelled: 'statusCancelled',
};

function statusTone(status: string): 'accent' | 'success' | 'warning' | 'danger' | 'neutral' {
  if (status === 'ready') return 'success';
  if (status === 'cancelled') return 'danger';
  if (status === 'awaiting_payment' || status === 'revision') return 'warning';
  if (status === 'generating' || status === 'paid') return 'accent';
  return 'neutral';
}

function statusLabel(t: (key: DictKey) => string, status: string): string {
  const key = STATUS_KEY[status];
  return key ? t(key) : status;
}

/** SruDoc space: the caller's orders from the existing DocGen service. */
export function DocHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const { status: session } = useSession();
  const status = useDocStatus(session === 'ready');
  const orders = useDocOrders(status.data === true);

  return (
    <Screen header={<AppHeader title="SruDoc" onBack={() => navigate('/apps')} leadingIcon="description" />}>
      {status.data === false ? (
        <div className="pt-2">
          <Banner tone="info" icon="link" title={t('docNotLinked')}>
            {t('docNotLinkedText')}
          </Banner>
        </div>
      ) : null}

      {status.data === true ? (
        <>
          <div className="flex flex-col gap-2 pt-3">
            {orders.isLoading ? <Skeleton className="h-20 w-full" /> : null}

            {orders.isError ? (
              <Banner tone="danger" icon="error" title={t('error')}>
                {t('docUnavailable')}
              </Banner>
            ) : null}

            {orders.data && orders.data.length === 0 ? (
              <EmptyState icon="description" title={t('docNoOrders')} text={t('docNoOrdersText')} />
            ) : null}

            {orders.data && orders.data.length > 0 ? (
              <Card>
                {orders.data.map((order, index) => (
                  <div key={order.number}>
                    {index > 0 ? <Hairline /> : null}
                    <Row
                      icon="description"
                      title={order.title || `${t('docOrder')} ${order.number}`}
                      subtitle={`№ ${order.number}`}
                      onClick={() => navigate(`/apps/doc/order/${order.number}`)}
                      trailing={
                        <Chip tone={statusTone(order.status)}>{statusLabel(t, order.status)}</Chip>
                      }
                    />
                  </div>
                ))}
              </Card>
            ) : null}
          </div>

          <div className="px-1 pt-4 text-caption text-ink-2">{t('docNote')}</div>
        </>
      ) : null}
    </Screen>
  );
}
