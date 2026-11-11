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
import { useT, type DictKey } from '@/shared/i18n';

const STATUS_KEY: Record<string, DictKey> = {
  draft: 'statusDraft',
  awaiting_payment: 'statusAwaitingPayment',
  paid: 'statusPaid',
  generating: 'statusGenerating',
  ready: 'statusReady',
  revision: 'statusRevision',
  cancelled: 'statusCancelled',
};

function statusLabel(t: (key: DictKey) => string, status: string): string {
  const key = STATUS_KEY[status];
  return key ? t(key) : status;
}

/** One SruDoc order, with the SruDoc -> SruDay deadline action. */
export function DocOrderPage() {
  const { number = '' } = useParams();
  const t = useT();
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
          feedback(t('docDeadlineAdded'));
        },
        onError: () => feedback(t('docDeadlineFailed'), 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title={`${t('docOrder')} ${number}`} onBack={() => navigate('/apps/doc')} />}>
      {order.isLoading ? <Skeleton className="mt-2 h-24 w-full" /> : null}

      {order.isError ? (
        <div className="pt-2">
          <Banner tone="danger" icon="error" title={t('docUnavailableOne')}>
            {t('docUnavailableOneText')}
          </Banner>
        </div>
      ) : null}

      {order.data ? (
        <div className="pt-2">
          <Card>
            <Row title={order.data.title || t('docUntitled')} subtitle={`№ ${order.data.number}`} />
            <Row
              title={t('docStatus')}
              value={<Chip tone="accent">{statusLabel(t, order.data.status)}</Chip>}
            />
            <Row title={t('docAmount')} value={`${order.data.amount_minor} ₸`} />
            <Row
              title={t('docResult')}
              subtitle={order.data.has_result ? t('docFileReady') : t('docFilePending')}
              icon={order.data.has_result ? 'download' : 'schedule'}
              iconTone={order.data.has_result ? 'success' : 'ink-2'}
            />
          </Card>

          {picking ? (
            <Card className="mt-3 p-3">
              <label className="text-caption text-ink-2" htmlFor="due-date">
                {t('docDueDate')}
              </label>
              <input
                id="due-date"
                type="date"
                value={dueDate}
                onChange={(event) => setDueDate(event.target.value)}
                className="mt-1 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none"
              />
              <div className="flex items-center justify-between pt-2">
                <button type="button" onClick={() => setPicking(false)} className="text-body-secondary text-ink-2">
                  {t('cancel')}
                </button>
                <PrimaryButton onClick={createDeadline} disabled={!dueDate} progress={deadline.isPending}>
                  {t('docCreateEvent')}
                </PrimaryButton>
              </div>
            </Card>
          ) : (
            <div className="pt-3">
              <SecondaryButton onClick={() => setPicking(true)} full>
                {t('docAddDeadline')}
              </SecondaryButton>
            </div>
          )}

          <div className="px-1 pt-3 text-caption text-ink-2">{t('docNote')}</div>
        </div>
      ) : null}
    </Screen>
  );
}
