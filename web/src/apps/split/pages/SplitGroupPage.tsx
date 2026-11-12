import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { formatMinor, parseMinor, useSplitBalances, useSplitExpenses } from '../api';
import { newRequestId, useAddExpense } from '../mutations';
import { usePeople } from '@/core/social/api';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  EmptyState,
  Hairline,
  Icon,
  PrimaryButton,
  Row,
  Screen,
  SecondaryButton,
  SectionLabel,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { useT, useI18n } from '@/shared/i18n';
import { formatDayMonth } from '@/shared/i18n/date';

/** One expense group: balances, expenses and the add form. */
export function SplitGroupPage() {
  const { groupId = '' } = useParams();
  const t = useT();
  const { lang } = useI18n();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const { status } = useSession();
  const ready = status === 'ready';
  const expenses = useSplitExpenses(groupId, ready);
  const balances = useSplitBalances(groupId, ready);
  const addExpense = useAddExpense(groupId);
  const people = usePeople(Boolean(groupId) && ready);

  const [composing, setComposing] = useState(false);
  const [description, setDescription] = useState('');
  const [amount, setAmount] = useState('');
  const [participants, setParticipants] = useState<string[]>([]);

  const nameOf = (userId: string): string => {
    if (userId === me?.id) return t('you');
    const person = people.data?.find((candidate) => candidate.id === userId);
    return person?.public_name ?? `${t('member')} ${userId}`;
  };

  const toggleParticipant = (userId: string) => {
    setParticipants((current) =>
      current.includes(userId) ? current.filter((value) => value !== userId) : [...current, userId],
    );
  };

  const submit = () => {
    const minor = parseMinor(amount);
    if (!description.trim() || minor === null || !me) {
      feedback(t('checkAmount'), 'error');
      return;
    }
    // The payer takes part by default: a share nobody carries would break the
    // zero-sum invariant the server checks.
    const chosen = participants.length > 0 ? participants : [me.id];
    addExpense.mutate(
      {
        clientRequestId: newRequestId(),
        description: description.trim(),
        amountMinor: minor,
        payerId: me.id,
        participants: chosen.includes(me.id) ? chosen : [me.id, ...chosen],
      },
      {
        onSuccess: () => {
          setDescription('');
          setAmount('');
          setParticipants([]);
          setComposing(false);
          feedback(t('expenseAdded'));
        },
        onError: () => feedback(t('expenseFailed'), 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title={t('splitGroup')} onBack={() => navigate('/apps/split')} />}>
      <div className="flex gap-2 pt-2">
        <SecondaryButton onClick={() => navigate(`/apps/split/${groupId}/receipt`)} full>
          <Icon name="photo_camera" size={18} />
          {t('receiptAdd')}
        </SecondaryButton>
        <SecondaryButton onClick={() => navigate(`/apps/split/${groupId}/debts`)} full>
          <Icon name="handshake" size={18} />
          {t('debts')}
        </SecondaryButton>
      </div>

      <SectionLabel>{t('balances')}</SectionLabel>
      <Card>
        {balances.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-10 w-full" />
          </div>
        ) : null}

        {balances.data && balances.data.length === 0 ? (
          <div className="px-4 py-3 text-caption text-ink-2">{t('noMembers')}</div>
        ) : null}

        {balances.data?.map((balance, index) => {
          const value = Number(balance.balance_minor);
          const mine = balance.user_id === me?.id;
          return (
            <div key={balance.user_id}>
              {index > 0 ? <Hairline /> : null}
              <Row
                icon="person"
                iconTone={mine ? 'accent' : 'ink-2'}
                title={nameOf(balance.user_id)}
                subtitle={`${t('paid')} ${formatMinor(balance.paid_minor)} ₸ · ${t('share2')} ${formatMinor(balance.share_minor)} ₸`}
                trailing={
                  <Chip tone={value > 0 ? 'success' : value < 0 ? 'danger' : 'neutral'}>
                    {value > 0 ? '+' : ''}
                    {formatMinor(balance.balance_minor)} ₸
                  </Chip>
                }
              />
            </div>
          );
        })}
      </Card>

      <SectionLabel>{t('expenses')}</SectionLabel>
      {composing ? (
        <Card className="p-3">
          <input
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder={t('expenseWhat')}
            maxLength={200}
            className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <input
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            placeholder={t('expenseAmount')}
            inputMode="decimal"
            className="mt-2 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />

          <div className="pt-3 text-caption text-ink-2">{t('addMembers')}</div>
          <div className="flex flex-wrap gap-1.5 pt-1">
            {people.data?.map((person) => {
              const picked = participants.includes(person.id);
              return (
                <button
                  key={person.id}
                  type="button"
                  onClick={() => toggleParticipant(person.id)}
                  className={
                    picked
                      ? 'h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                      : 'h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                  }
                >
                  {person.public_name}
                </button>
              );
            })}
          </div>

          <div className="flex items-center justify-between pt-3">
            <button type="button" onClick={() => setComposing(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
            <PrimaryButton onClick={submit} progress={addExpense.isPending}>
              {t('save')}
            </PrimaryButton>
          </div>
        </Card>
      ) : (
        <button
          type="button"
          onClick={() => setComposing(true)}
          className="flex w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint"
        >
          <Icon name="add_circle" size={20} className="text-accent" />
          {t('addExpense')}
        </button>
      )}

      <div className="pt-2">
        {expenses.isLoading ? <Skeleton className="h-16 w-full" /> : null}

        {expenses.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            {t('retryConnection')}
          </Banner>
        ) : null}

        {expenses.data && expenses.data.length === 0 ? (
          <EmptyState icon="receipt_long" title={t('expenseEmpty')} text={t('expenseEmptyText')} />
        ) : null}

        {expenses.data && expenses.data.length > 0 ? (
          <Card>
            {expenses.data.map((expense, index) => (
              <div key={expense.id}>
                {index > 0 ? <Hairline /> : null}
                <Row
                  icon="payments"
                  title={expense.description || t('expenses')}
                  subtitle={formatDayMonth(expense.spent_at, lang)}
                  value={`${formatMinor(expense.amount_minor)} ₸`}
                />
              </div>
            ))}
          </Card>
        ) : null}
      </div>

      <div className="px-1 pt-4 text-caption text-ink-2">{t('splitNote')}</div>
    </Screen>
  );
}
