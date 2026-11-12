import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { chatKeys, useGroupMembers, type GroupMember } from './api';
import { useAddGroupMember, useRemoveGroupMember } from './mutations';
import { usePeople } from '@/core/social/api';
import { useSession } from '@/app/session';
import { ApiError } from '@/shared/api/client';
import { useT } from '@/shared/i18n';
import { showConfirm } from '@/shared/telegram/sdk';
import { AppHeader, Avatar, Banner, Card, EmptyState, Hairline, Icon, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';

export function GroupMembers({ conversationId, onClose }: { conversationId: string; onClose: () => void }) {
  const t = useT();
  const { me, status } = useSession();
  const feedback = useFeedback();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [busy, setBusy] = useState(false);
  const dialog = useRef<HTMLDialogElement>(null);
  const acting = useRef(false);
  const members = useGroupMembers(conversationId, status === 'ready');
  const owner = members.data?.find((member) => member.user_id === me?.id)?.role === 'owner';
  const count = members.data?.length ?? 0;
  const people = usePeople(status === 'ready' && owner && count < 50, search.trim());
  const add = useAddGroupMember(conversationId);
  const remove = useRemoveGroupMember(conversationId);
  const shownPeople = people.data?.filter((person) => !members.data?.some((member) => member.user_id === person.id));

  useEffect(() => {
    const element = dialog.current;
    element?.showModal();
    return () => element?.close();
  }, []);

  const fail = (error: unknown) => {
    const code = error instanceof ApiError ? error.code : '';
    feedback(t(code === 'owner_must_remove_members' ? 'chatGroupOwnerLeaveHint' : code === 'group_full' ? 'chatGroupFull' : 'chatGroupActionFailed'), 'error');
  };

  const addMember = async (userId: string) => {
    if (acting.current) return;
    acting.current = true;
    setBusy(true);
    try {
      await add.mutateAsync(userId);
      setSearch('');
      feedback(t('chatGroupMemberAdded'));
    } catch (error) { fail(error); }
    finally { acting.current = false; setBusy(false); }
  };

  const removeMember = async (member: GroupMember) => {
    if (acting.current) return;
    acting.current = true;
    setBusy(true);
    const self = member.user_id === me?.id;
    try {
      if (!await showConfirm(self ? t('chatGroupLeaveConfirm') : `${t('chatGroupRemoveConfirm')}\n${member.public_name}`)) return;
      await remove.mutateAsync(member.user_id);
      if (self) {
        await queryClient.cancelQueries({ queryKey: chatKeys.history(conversationId) });
        queryClient.removeQueries({ queryKey: chatKeys.history(conversationId) });
        queryClient.removeQueries({ queryKey: chatKeys.members(conversationId) });
        navigate('/chats', { replace: true });
      } else feedback(t('chatGroupMemberRemoved'));
    } catch (error) { fail(error); }
    finally { acting.current = false; setBusy(false); }
  };

  const self = members.data?.find((member) => member.user_id === me?.id);

  return <dialog ref={dialog} onCancel={(event) => { event.preventDefault(); onClose(); }}
    aria-label={t('chatGroupMembers')} className="fixed inset-0 m-0 h-full max-h-none w-full max-w-none overflow-y-auto bg-page p-0 text-ink backdrop:bg-black/30">
    <Screen header={<AppHeader title={t('chatGroupMembers')} trailing={<>
      <button type="button" onClick={() => void members.refetch()} disabled={members.isFetching || busy} aria-label={t('refresh')} className="flex size-11 items-center justify-center text-accent disabled:opacity-50"><Icon name="refresh" size={22} /></button>
      <button type="button" onClick={onClose} aria-label={t('close')} className="flex size-11 items-center justify-center text-accent"><Icon name="close" size={24} /></button>
    </>} />}>
      <div className="flex flex-col gap-3 py-3">
        {members.isLoading ? <Skeleton className="h-28 w-full" /> : null}
        {members.isError ? <Banner tone="danger" icon="error" title={t('error')}><button type="button" onClick={() => void members.refetch()} className="min-h-11 text-accent">{t('retry')}</button></Banner> : null}
        {members.data ? <>
          <p className="px-1 text-caption text-ink-2">{t('chatGroupMembers')} · {count}/50</p>
          <Card>
            {members.data.map((member, index) => <div key={member.user_id}>
              {index ? <Hairline /> : null}
              <div className="flex min-h-16 items-center gap-3 px-4 py-3">
                <Avatar name={member.public_name} size={40} />
                <span className="min-w-0 flex-1">
                  <span className="block break-words text-body-strong [overflow-wrap:anywhere]">{member.public_name}{member.user_id === me?.id ? ` · ${t('you')}` : ''}</span>
                  <span className="text-caption text-ink-2">{t(member.role === 'owner' ? 'chatGroupOwner' : 'member')}</span>
                </span>
                {owner && member.user_id !== me?.id ? <button type="button" disabled={busy} onClick={() => void removeMember(member)} aria-label={`${t('delete')}: ${member.public_name}`} className="flex size-11 shrink-0 items-center justify-center text-danger disabled:opacity-50"><Icon name="person_remove" size={22} /></button> : null}
              </div>
            </div>)}
          </Card>
          {owner ? <>
            <p className="px-1 text-caption text-ink-2">{t('chatGroupHint')}</p>
            {count >= 50 ? <p className="px-1 text-body-secondary text-ink-2">{t('chatGroupFull')}</p> : <>
              <label className="flex items-center gap-2 rounded-xl bg-surface px-3 text-ink-2"><Icon name="person_add" size={22} />
                <input value={search} onChange={(event) => setSearch(event.target.value)} disabled={busy} aria-label={t('chatGroupAdd')} placeholder={t('peopleSearch')} className="h-12 min-w-0 flex-1 bg-transparent text-body-lg text-ink outline-none placeholder:text-hint disabled:opacity-50" />
              </label>
              {people.isLoading ? <Skeleton className="h-20 w-full" /> : null}
              {people.isError ? <Banner tone="danger" icon="error" title={t('error')}><button type="button" onClick={() => void people.refetch()} className="min-h-11 text-accent">{t('retry')}</button></Banner> : null}
              <Card>{shownPeople?.map((person, index) => <div key={person.id}>
                {index ? <Hairline /> : null}
                <button type="button" disabled={busy} onClick={() => void addMember(person.id)} aria-label={`${t('chatGroupAdd')}: ${person.public_name}`} className="flex min-h-16 w-full items-center gap-3 px-4 py-3 text-left disabled:opacity-50">
                  <Avatar name={person.public_name} size={36} /><span className="min-w-0 flex-1 break-words text-body-strong [overflow-wrap:anywhere]">{person.public_name}</span><Icon name="add" size={22} className="text-accent" />
                </button>
              </div>)}{shownPeople?.length === 0 ? <EmptyState icon="person_search" title={t('empty')} text={t('noSearchResults')} /> : null}</Card>
            </>}
          </> : null}
          {self ? <div className="pt-2">
            {owner && count > 1 ? <p className="px-1 pb-2 text-caption text-ink-2">{t('chatGroupOwnerLeaveHint')}</p> : null}
            <button type="button" disabled={busy || (owner && count > 1)} onClick={() => void removeMember(self)} className="flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-surface px-4 text-body-strong text-danger disabled:opacity-50"><Icon name="logout" size={22} />{t('chatGroupLeave')}</button>
          </div> : null}
        </> : null}
      </div>
    </Screen>
  </dialog>;
}
