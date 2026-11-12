import { useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCreateGroup } from './mutations';
import { usePeople, type Person } from '@/core/social/api';
import { AppHeader, Avatar, Banner, Card, EmptyState, Hairline, Icon, PrimaryButton, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { ApiError } from '@/shared/api/client';
import { useT } from '@/shared/i18n';

export function CreateGroupPage() {
  const t = useT();
  const feedback = useFeedback();
  const navigate = useNavigate();
  const { status } = useSession();
  const [title, setTitle] = useState('');
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState<Person[]>([]);
  const people = usePeople(status === 'ready', search.trim());
  const create = useCreateGroup();
  const submitting = useRef(false);
  const attempt = useRef<{ snapshot: string; id: string } | null>(null);

  const toggle = (person: Person) => setSelected((current) => {
    if (current.some((item) => item.id === person.id)) return current.filter((item) => item.id !== person.id);
    return current.length < 49 ? [...current, person] : current;
  });

  const submit = () => {
    if (!title.trim() || selected.length < 1 || status !== 'ready' || submitting.current) return;
    submitting.current = true;
    const userIds = selected.map((person) => person.id).sort();
    const snapshot = JSON.stringify([title.trim(), userIds]);
    if (attempt.current?.snapshot !== snapshot) attempt.current = { snapshot, id: crypto.randomUUID() };
    create.mutate({ client_group_id: attempt.current.id, title: title.trim(), user_ids: userIds }, {
      onSuccess: (group) => { feedback(t('groupCreated')); navigate(`/chats/${group.id}`, { replace: true }); },
      onError: (error) => feedback(t(error instanceof ApiError && error.code === 'idempotency_conflict' ? 'chatGroupCreateConflict' : 'groupFailed'), 'error'),
      onSettled: () => { submitting.current = false; },
    });
  };

  return (
    <Screen header={<AppHeader title={t('chatNewGroup')} onBack={() => navigate('/chats')} />} bottomSpace>
      <div className="flex flex-col gap-3 pt-3">
        <Card className="gap-2 p-4">
          <label className="text-body-secondary-strong text-ink" htmlFor="chat-group-name">{t('chatGroupName')}</label>
          <input id="chat-group-name" value={title} onChange={(event) => setTitle(event.target.value)} maxLength={100} disabled={create.isPending}
            className="h-12 rounded-xl bg-neutral-soft/70 px-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent disabled:opacity-50" />
          <p className="text-caption text-ink-2">{t('chatGroupHint')}</p>
        </Card>
        <div className="flex items-center justify-between px-1 text-body-secondary text-ink-2"><span>{t('chatGroupSelected')}</span><span>{selected.length}/49</span></div>
        {selected.length > 0 ? <div className="flex max-h-32 flex-wrap gap-2 overflow-y-auto">
          {selected.map((person) => <button key={person.id} type="button" onClick={() => toggle(person)} disabled={create.isPending}
            aria-label={`${t('delete')}: ${person.public_name}`} className="flex min-h-11 max-w-full items-center gap-1 rounded-full bg-accent-soft px-3 text-body-secondary text-accent disabled:opacity-50">
            <span className="truncate">{person.public_name}</span><Icon name="close" size={18} />
          </button>)}
        </div> : null}
        <label className="flex items-center gap-2 rounded-xl bg-surface px-3 text-ink-2">
          <Icon name="search" size={22} />
          <input value={search} onChange={(event) => setSearch(event.target.value)} disabled={create.isPending} aria-label={t('peopleSearch')} placeholder={t('peopleSearch')}
            className="h-12 min-w-0 flex-1 bg-transparent text-body-lg text-ink outline-none placeholder:text-hint disabled:opacity-50" />
        </label>
        {people.isLoading ? <Skeleton className="h-28 w-full" /> : null}
        {people.isError ? <Banner tone="danger" icon="error" title={t('error')}><button type="button" onClick={() => void people.refetch()} className="min-h-11 text-accent">{t('retry')}</button></Banner> : null}
        <Card>
          {people.data?.map((person, index) => {
            const checked = selected.some((item) => item.id === person.id);
            return <div key={person.id}>
              {index ? <Hairline /> : null}
              <button type="button" aria-pressed={checked} onClick={() => toggle(person)} disabled={create.isPending || (!checked && selected.length >= 49)}
                className="flex min-h-16 w-full items-center gap-3 px-4 py-3 text-left disabled:opacity-50">
                <Avatar name={person.public_name} size={40} />
                <span className="min-w-0 flex-1 break-words text-body-strong text-ink [overflow-wrap:anywhere]">{person.public_name}</span>
                <Icon name={checked ? 'check_circle' : 'radio_button_unchecked'} size={24} className={checked ? 'text-accent' : 'text-ink-2'} />
              </button>
            </div>;
          })}
          {people.data?.length === 0 ? <EmptyState icon="person_search" title={t('empty')} text={t(search.trim() ? 'noSearchResults' : 'peopleEmpty')} /> : null}
        </Card>
      </div>
      <div className="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-surface/95 backdrop-blur-xl" style={{ paddingBottom: 'var(--safe-bottom)' }}>
        <div className="mx-auto max-w-[480px] px-4 py-3"><PrimaryButton full onClick={submit} disabled={!title.trim() || selected.length < 1 || status !== 'ready'} progress={create.isPending}>{t('chatNewGroup')}</PrimaryButton></div>
      </div>
    </Screen>
  );
}
