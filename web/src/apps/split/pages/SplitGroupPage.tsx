import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { formatMinor, parseMinor, useSplitBalances, useSplitExpenses } from '../api';
import { newRequestId, useAddExpense } from '../mutations';
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
  SectionLabel,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { useT } from '@/shared/i18n';

/** One expense group: balances, expenses and the add form. */
export function SplitGroupPage() {
  const { id = '' } = useParams();
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const expenses = useSplitExpenses(id);
  const balances = useSplitBalances(id);
  const addExpense = useAddExpense(id);

  const [composing, setComposing] = useState(false);
  const [description, setDescription] = useState('');
  const [amount, setAmount] = useState('');

  const submit = () => {
    const minor = parseMinor(amount);
    if (!description.trim() || minor === null || !me) {
      feedback('Проверьте сумму и описание', 'error');
      return;
    }
    addExpense.mutate(
      {
        clientRequestId: newRequestId(),
        description: description.trim(),
        amountMinor: minor,
        payerId: me.id,
        participants: [me.id],
      },
      {
        onSuccess: () => {
          setDescription('');
          setAmount('');
          setComposing(false);
          feedback('Расход добавлен');
        },
        onError: () => feedback('Не удалось добавить расход', 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title="Группа расходов" onBack={() => navigate('/apps/split')} />}>
      <SectionLabel>Балансы</SectionLabel>
      <Card>
        {balances.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-10 w-full" />
          </div>
        ) : null}

        {balances.data && balances.data.length === 0 ? (
          <div className="px-4 py-3 text-caption text-ink-2">Пока никого нет в группе</div>
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
                title={mine ? 'Вы' : `Участник ${balance.user_id}`}
                subtitle={`Оплачено ${formatMinor(balance.paid_minor)} ₸ · доля ${formatMinor(balance.share_minor)} ₸`}
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

      <SectionLabel>Расходы</SectionLabel>
      {composing ? (
        <Card className="p-3">
          <input
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder="На что потратили"
            maxLength={200}
            className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <input
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            placeholder="Сумма, ₸"
            inputMode="decimal"
            className="mt-2 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <div className="flex items-center justify-between pt-2">
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
          Добавить расход
        </button>
      )}

      <div className="pt-2">
        {expenses.isLoading ? <Skeleton className="h-16 w-full" /> : null}

        {expenses.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            Проверьте соединение и повторите.
          </Banner>
        ) : null}

        {expenses.data && expenses.data.length === 0 ? (
          <EmptyState icon="receipt_long" title="Расходов пока нет" text="Добавьте первый расход группы." />
        ) : null}

        {expenses.data && expenses.data.length > 0 ? (
          <Card>
            {expenses.data.map((expense, index) => (
              <div key={expense.id}>
                {index > 0 ? <Hairline /> : null}
                <Row
                  icon="payments"
                  title={expense.description || 'Расход'}
                  subtitle={new Date(expense.spent_at).toLocaleDateString('ru-RU', {
                    day: 'numeric',
                    month: 'short',
                  })}
                  value={`${formatMinor(expense.amount_minor)} ₸`}
                />
              </div>
            ))}
          </Card>
        ) : null}
      </div>

      <div className="px-1 pt-4 text-caption text-ink-2">
        Сумма делится поровну, остаток округления достаётся первым участникам по порядку. Сумма
        балансов группы всегда равна нулю.
      </div>
    </Screen>
  );
}
