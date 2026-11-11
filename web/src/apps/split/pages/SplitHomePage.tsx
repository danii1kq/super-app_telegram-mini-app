import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useSplitGroups } from '../api';
import { useCreateGroup } from '../mutations';
import {
  AppHeader,
  Banner,
  Card,
  EmptyState,
  Hairline,
  Icon,
  PrimaryButton,
  Row,
  Screen,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';

/** SruSplit: expense groups. */
export function SplitHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const groups = useSplitGroups();
  const createGroup = useCreateGroup();
  const [composing, setComposing] = useState(false);
  const [title, setTitle] = useState('');

  const submit = () => {
    const value = title.trim();
    if (!value) return;
    createGroup.mutate(
      { title: value, members: [] },
      {
        onSuccess: (group) => {
          setTitle('');
          setComposing(false);
          feedback('Группа создана');
          navigate(`/apps/split/${group.id}`);
        },
        onError: () => feedback('Не удалось создать группу', 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title="SruSplit" onBack={() => navigate('/apps')} leadingIcon="receipt_long" />}>
      {composing ? (
        <Card className="mt-2 p-3">
          <input
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder="Название группы, например «Кафе»"
            maxLength={120}
            className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <div className="flex items-center justify-between pt-2">
            <button type="button" onClick={() => setComposing(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
            <PrimaryButton onClick={submit} disabled={!title.trim()} progress={createGroup.isPending}>
              {t('save')}
            </PrimaryButton>
          </div>
        </Card>
      ) : (
        <button
          type="button"
          onClick={() => setComposing(true)}
          className="mt-2 flex w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint"
        >
          <Icon name="group_add" size={20} className="text-accent" />
          Новая группа расходов
        </button>
      )}

      <div className="pt-3">
        {groups.isLoading ? <Skeleton className="h-16 w-full" /> : null}

        {groups.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            Проверьте соединение и повторите.
          </Banner>
        ) : null}

        {groups.data && groups.data.length === 0 ? (
          <EmptyState
            icon="receipt_long"
            title={t('empty')}
            text="Создайте группу, добавьте расходы, и SruSplit посчитает, кто кому сколько должен."
          />
        ) : null}

        {groups.data && groups.data.length > 0 ? (
          <Card>
            {groups.data.map((group, index) => (
              <div key={group.id}>
                {index > 0 ? <Hairline /> : null}
                <Row
                  icon="receipt_long"
                  title={group.title}
                  subtitle={new Date(group.created_at).toLocaleDateString('ru-RU', {
                    day: 'numeric',
                    month: 'short',
                  })}
                  onClick={() => navigate(`/apps/split/${group.id}`)}
                />
              </div>
            ))}
          </Card>
        ) : null}
      </div>

      <div className="px-1 pt-4 text-caption text-ink-2">
        Расчёты в тенге, без банковских переводов. Суммы видят только участники группы.
      </div>
    </Screen>
  );
}
