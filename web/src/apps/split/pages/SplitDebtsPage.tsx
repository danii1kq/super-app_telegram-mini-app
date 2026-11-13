import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { formatMinor, parseMinor } from '../api';
import { newRequestId } from '../mutations';
import { useAddPayment, useCreateDebt, useDebts, useGroupMembers, useGroupPeople, useRespondPayment } from '../receipts';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  EmptyState,
  PrimaryButton,
  Screen,
  SecondaryButton,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { ApiError } from '@/shared/api/client';
import { useT, useI18n } from '@/shared/i18n';
import { formatDayMonth } from '@/shared/i18n/date';

/**
 * Debts of one group: who owes whom, with repayments.
 *
 * A debt that points at an expense is a handle for repayments: the expense
 * already carries the money, so recording it never adds the amount again.
 */
export function SplitDebtsPage() {
  const { groupId = '' } = useParams();
  const t = useT();
  const { lang } = useI18n();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const debts = useDebts(groupId);
  const people = useGroupPeople(groupId);
  const groupMembers = useGroupMembers(groupId);
  const createDebt = useCreateDebt(groupId);
  const addPayment = useAddPayment(groupId);
  const respondPayment=useRespondPayment(groupId);

  const [composing, setComposing] = useState(false);
  const [creditorId, setCreditorId] = useState('');
  const [debtorId, setDebtorId] = useState('');
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [requestKey, setRequestKey] = useState(newRequestId);
  const [payingId, setPayingId] = useState('');
  const [paymentAmount, setPaymentAmount] = useState('');
  const [paymentKey, setPaymentKey] = useState(newRequestId);

  // Only the group's own members can be a side of a debt here, and the viewer
  // is one of them: the directory leaves the viewer out.
  const members = (() => {
    const ids = new Set(groupMembers.data ?? []);
    const fromDirectory = (people.data ?? []).filter((person) => ids.has(person.id));
    if (me && ids.has(me.id) && !fromDirectory.some((person) => person.id === me.id)) {
      return [{ id: me.id, public_name: me.public_name, username: me.username, following: false, is_me: true }, ...fromDirectory];
    }
    return fromDirectory;
  })();
  const nameOf = (userId: string): string => {
    if (userId === me?.id) return t('you');
    return members.find((person) => person.id === userId)?.public_name ?? `${t('member')} ${userId}`;
  };

  const submit = () => {
    const minor = parseMinor(amount);
    if (!creditorId || !debtorId || minor === null) {
      feedback(t('checkAmount'), 'error');
      return;
    }
    if (creditorId === debtorId) {
      feedback(t('debtSamePerson'), 'error');
      return;
    }
    createDebt.mutate(
      {
        clientRequestId: requestKey,
        creditorId,
        debtorId,
        amountMinor: String(minor),
        note: note.trim(),
      },
      {
        onSuccess: () => {
          setAmount('');
          setNote('');
          setComposing(false);
          setRequestKey(newRequestId());
          feedback(t('debtCreated'));
        },
        onError: () => feedback(t('debtFailed'), 'error'),
      },
    );
  };

  const pay = (debtId: string, outstanding: string) => {
    const minor = parseMinor(paymentAmount);
    if (minor === null) {
      feedback(t('checkAmount'), 'error');
      return;
    }
    if (minor > Number(outstanding)) {
      feedback(t('debtTooMuch'), 'error');
      return;
    }
    addPayment.mutate(
      { debtId, amountMinor: String(minor), clientRequestId: paymentKey },
      {
        onSuccess: (debt) => {
          setPayingId('');
          setPaymentAmount('');
          setPaymentKey(newRequestId());
          feedback(t(debt.payments.some(p=>p.status==='pending')?'splitPaymentAwaiting':'debtPaid'));
        },
        onError: (error) => feedback(error instanceof ApiError?error.message:t('debtPayFailed'), 'error'),
      },
    );
  };

  return (
    <Screen
      header={<AppHeader title={t('debts')} onBack={() => navigate(`/apps/split/${groupId}`)} />}
    >
      {composing ? (
        <Card className="mt-2 p-3">
          <div className="text-caption text-ink-2">{t('debtCreditor')}</div>
          <div className="flex flex-wrap gap-1.5 pt-1">
            {members.map((person) => (
              <button
                key={person.id}
                type="button"
                onClick={() => setCreditorId(person.id)}
                className={
                  creditorId === person.id
                    ? 'h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                    : 'h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                }
              >
                {person.public_name}
              </button>
            ))}
          </div>

          <div className="pt-3 text-caption text-ink-2">{t('debtDebtor')}</div>
          <div className="flex flex-wrap gap-1.5 pt-1">
            {members.map((person) => (
              <button
                key={person.id}
                type="button"
                onClick={() => setDebtorId(person.id)}
                className={
                  debtorId === person.id
                    ? 'h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                    : 'h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                }
              >
                {person.public_name}
              </button>
            ))}
          </div>

          <input
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            placeholder={t('expenseAmount')}
            inputMode="decimal"
            className="mt-3 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <input
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder={t('debtNote')}
            maxLength={200}
            className="mt-2 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />

          <div className="flex items-center justify-between pt-2">
            <button type="button" onClick={() => setComposing(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
            <PrimaryButton onClick={submit} progress={createDebt.isPending}>
              {t('save')}
            </PrimaryButton>
          </div>
        </Card>
      ) : (
        <div className="pt-2">
          <SecondaryButton onClick={() => setComposing(true)} full>
            {t('debtNew')}
          </SecondaryButton>
        </div>
      )}

      <div className="pt-3">
        {debts.isLoading ? <Skeleton className="h-20 w-full" /> : null}

        {debts.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            {t('retryConnection')}
          </Banner>
        ) : null}

        {debts.data && debts.data.length === 0 ? (
          <EmptyState icon="handshake" title={t('debtEmpty')} text={t('debtEmptyText')} />
        ) : null}

        <div className="flex flex-col gap-2">
          {debts.data?.map((debt) => {
            const outstanding = Number(debt.outstanding_minor);
            const settled = outstanding <= 0;
            const canPay = me && (me.id === debt.creditor_id || me.id === debt.debtor_id);
            return (
              <Card key={debt.id} className="p-4">
                <div className="flex items-center justify-between">
                  <span className="text-body-strong text-ink">
                    {nameOf(debt.debtor_id)} → {nameOf(debt.creditor_id)}
                  </span>
                  <Chip tone={settled ? 'success' : 'warning'}>
                    {settled ? t('debtSettled') : `${formatMinor(debt.outstanding_minor)} ₸`}
                  </Chip>
                </div>

                <div className="pt-1 text-caption text-ink-2">
                  {t('debtAmount')}: {formatMinor(debt.amount_minor)} ₸
                  {debt.source_expense_id ? ` · ${t('debtFromExpense')}` : ` · ${t('debtManual')}`}
                </div>
                {debt.note ? <div className="pt-1 text-body-secondary text-ink">{debt.note}</div> : null}
                <div className="pt-1 text-caption text-ink-2">
                  {t('debtAuthor')}: {nameOf(debt.author_id)}
                </div>

                {debt.payments.length > 0 ? (
                  <div className="pt-2">
                    <div className="text-caption text-ink-2">{t('debtHistory')}</div>
                    {debt.payments.map((payment) => (
                      <div key={payment.id} className="flex flex-wrap items-center justify-between gap-2 pt-2">
                        <span className="text-caption text-ink-2">
                          {formatDayMonth(payment.paid_at, lang)}
                        </span>
                        <span className={`text-caption ${payment.status==='confirmed'?'text-success':'text-ink-2'}`}>
                          {formatMinor(payment.amount_minor)} ₸ · {t(payment.status==='confirmed'?'splitPaymentConfirmed':payment.status==='pending'?'splitPaymentAwaiting':'splitPaymentRejected')}
                        </span>
                        {payment.status==='pending'&&debt.creditor_id===me?.id?<div className="flex w-full gap-2"><SecondaryButton disabled={respondPayment.isPending} onClick={()=>respondPayment.mutate({debtId:debt.id,paymentId:payment.id,accept:true},{onError:e=>feedback(e instanceof ApiError?e.message:t('debtPayFailed'),'error')})}>{t('splitReceived')}</SecondaryButton><SecondaryButton disabled={respondPayment.isPending} onClick={()=>respondPayment.mutate({debtId:debt.id,paymentId:payment.id,accept:false},{onError:e=>feedback(e instanceof ApiError?e.message:t('debtPayFailed'),'error')})}>{t('splitNotReceived')}</SecondaryButton></div>:null}
                      </div>
                    ))}
                  </div>
                ) : null}

                {!settled && canPay ? (
                  payingId === debt.id ? (
                    <div className="pt-3">
                      <input
                        value={paymentAmount}
                        onChange={(event) => setPaymentAmount(event.target.value)}
                        placeholder={t('debtPaymentAmount')}
                        inputMode="decimal"
                        className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
                      />
                      <div className="flex items-center justify-between pt-2">
                        <button
                          type="button"
                          onClick={() => setPayingId('')}
                          className="text-body-secondary text-ink-2"
                        >
                          {t('cancel')}
                        </button>
                        <div className="flex gap-2">
                          <button
                            type="button"
                            onClick={() => setPaymentAmount(formatMinor(debt.outstanding_minor))}
                            className="text-body-secondary text-accent"
                          >
                            {t('debtPayFull')}
                          </button>
                          <PrimaryButton
                            onClick={() => pay(debt.id, debt.outstanding_minor)}
                            progress={addPayment.isPending}
                          >
                            {t(debt.creditor_id===me?.id?'splitReceived':'splitSent')}
                          </PrimaryButton>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <div className="pt-3">
                      <SecondaryButton onClick={() => setPayingId(debt.id)} full>
                        {t(debt.creditor_id===me?.id?'splitReceived':'splitSent')}
                      </SecondaryButton>
                    </div>
                  )
                ) : null}
              </Card>
            );
          })}
        </div>
      </div>

      <div className="px-1 pt-4 text-caption text-ink-2">{t('debtNoteHint')}</div>
    </Screen>
  );
}
