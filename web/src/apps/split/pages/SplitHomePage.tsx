import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useSplitGroups } from '../api';
import { useCreateGroup } from '../mutations';
import { usePeople } from '@/core/social/api';
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
import { useT, useI18n } from '@/shared/i18n';
import { formatDayMonth } from '@/shared/i18n/date';

/** SruSplit: expense groups. */
export function SplitHomePage() {
  const t = useT();
  const { lang } = useI18n();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const groups = useSplitGroups();
  const createGroup = useCreateGroup();
  const [composing, setComposing] = useState(false);
  const [title, setTitle] = useState('');
  const [members, setMembers] = useState<string[]>([]);
  const people = usePeople(composing);

  const toggleMember = (id: string) => {
    setMembers((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    );
  };

  const submit = () => {
    const value = title.trim();
    if (!value) return;
    createGroup.mutate(
      { title: value, members },
      {
        onSuccess: (group) => {
          setTitle('');
          setMembers([]);
          setComposing(false);
          feedback(t('groupCreated'));
          navigate(`/apps/split/${group.id}`);
        },
        onError: () => feedback(t('groupFailed'), 'error'),
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
            placeholder={t('splitGroupTitle')}
            maxLength={120}
            className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />

          <div className="pt-3 text-caption text-ink-2">{t('addMembers')}</div>
          {people.data && people.data.length === 0 ? (
            <div className="py-2 text-caption text-ink-2">{t('peopleEmpty')}</div>
          ) : null}
          <div className="flex flex-wrap gap-1.5 pt-1">
            {people.data?.map((person) => {
              const picked = members.includes(person.id);
              return (
                <button
                  key={person.id}
                  type="button"
                  onClick={() => toggleMember(person.id)}
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
          {t('splitNew')}
        </button>
      )}

      <div className="pt-3">
        {groups.isLoading ? <Skeleton className="h-16 w-full" /> : null}

        {groups.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            {t('retryConnection')}
          </Banner>
        ) : null}

        {groups.data && groups.data.length === 0 ? (
          <EmptyState icon="receipt_long" title={t('empty')} text={t('splitEmpty')} />
        ) : null}

        {groups.data && groups.data.length > 0 ? (
          <Card>
            {groups.data.map((group, index) => (
              <div key={group.id}>
                {index > 0 ? <Hairline /> : null}
                <Row
                  icon="receipt_long"
                  title={group.title}
                  subtitle={formatDayMonth(group.created_at, lang)}
                  onClick={() => navigate(`/apps/split/${group.id}`)}
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
