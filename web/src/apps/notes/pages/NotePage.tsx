import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useNote } from '../api';
import { useDeleteNote, useGrantNote, useUpdateNote } from '../mutations';
import { usePeople } from '@/core/social/api';
import {
  AppHeader,
  Banner,
  Card,
  Hairline,
  Icon,
  PrimaryButton,
  Row,
  Screen,
  SecondaryButton,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';

/** One note: read it, edit it and share it when the viewer is the author. */
export function NotePage() {
  const { id = '' } = useParams();
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const note = useNote(id);
  const updateNote = useUpdateNote(id);
  const deleteNote = useDeleteNote();
  const grantNote = useGrantNote(id);
  const people = usePeople(Boolean(note.data?.mine));

  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [editing, setEditing] = useState(false);
  const [sharing, setSharing] = useState(false);

  useEffect(() => {
    if (!note.data) return;
    setTitle(note.data.title);
    setBody(note.data.body ?? '');
  }, [note.data]);

  const save = () => {
    if (!note.data) return;
    updateNote.mutate(
      { title, body, version: note.data.version },
      {
        onSuccess: () => {
          setEditing(false);
          feedback(t('saved'));
        },
        onError: () => feedback(t('noteConflict'), 'error'),
      },
    );
  };

  const remove = () => {
    deleteNote.mutate(id, {
      onSuccess: () => {
        feedback(t('noteDeleted'));
        navigate('/apps/notes');
      },
      onError: () => feedback(t('noteDeleteFailed'), 'error'),
    });
  };

  const share = (userId: string) => {
    grantNote.mutate([userId], {
      onSuccess: () => {
        setSharing(false);
        feedback(t('shareDone'));
      },
      onError: () => feedback(t('shareFailed'), 'error'),
    });
  };

  return (
    <Screen header={<AppHeader title={t('noteTitle')} onBack={() => navigate('/apps/notes')} />}>
      {note.isLoading ? <Skeleton className="mt-2 h-32 w-full" /> : null}

      {note.isError ? (
        <div className="pt-2">
          <Banner tone="danger" icon="lock" title={t('noteUnavailable')}>
            {t('noteUnavailableText')}
          </Banner>
        </div>
      ) : null}

      {note.data ? (
        <div className="pt-2">
          {editing ? (
            <Card className="p-3">
              <input
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder={t('noteTitle')}
                className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-strong text-ink outline-none placeholder:text-hint"
              />
              <textarea
                value={body}
                onChange={(event) => setBody(event.target.value)}
                rows={10}
                className="mt-2 w-full resize-none rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none"
              />
              <div className="flex items-center justify-between pt-2">
                <button type="button" onClick={() => setEditing(false)} className="text-body-secondary text-ink-2">
                  {t('cancel')}
                </button>
                <PrimaryButton onClick={save} progress={updateNote.isPending}>
                  {t('save')}
                </PrimaryButton>
              </div>
            </Card>
          ) : (
            <>
              <Card className="p-4">
                <h2 className="text-headline-md text-ink">{note.data.title || t('noteUntitled')}</h2>
                <p className="whitespace-pre-wrap pt-2 text-body-lg text-ink">{note.data.body}</p>
              </Card>

              {note.data.mine ? (
                <>
                  <div className="flex gap-2 pt-3">
                    <SecondaryButton onClick={() => setEditing(true)} full>
                      {t('edit')}
                    </SecondaryButton>
                    <SecondaryButton onClick={() => setSharing((value) => !value)} full>
                      {t('share')}
                    </SecondaryButton>
                  </div>
                  <div className="pt-2">
                    <SecondaryButton onClick={remove} full>
                      {t('delete')}
                    </SecondaryButton>
                  </div>

                  {sharing ? (
                    <Card className="mt-3">
                      <div className="px-4 pt-3 text-caption text-ink-2">{t('shareWith')}</div>
                      {people.data && people.data.length === 0 ? (
                        <div className="px-4 py-3 text-caption text-ink-2">{t('peopleEmpty')}</div>
                      ) : null}
                      {people.data?.map((person, index) => (
                        <div key={person.id}>
                          {index > 0 ? <Hairline /> : null}
                          <Row
                            title={person.public_name}
                            subtitle={person.username ? `@${person.username}` : undefined}
                            onClick={() => share(person.id)}
                            trailing={<Icon name="person_add" size={20} className="text-accent" />}
                          />
                        </div>
                      ))}
                    </Card>
                  ) : null}
                </>
              ) : (
                <div className="px-1 pt-3 text-caption text-ink-2">{t('noteSharedNote')}</div>
              )}
            </>
          )}
        </div>
      ) : null}
    </Screen>
  );
}
