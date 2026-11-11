import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useNote } from '../api';
import { useDeleteNote, useUpdateNote } from '../mutations';
import { AppHeader, Banner, Card, PrimaryButton, Screen, SecondaryButton, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';

/** One note: read it, and edit it when the viewer is the author. */
export function NotePage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const note = useNote(id);
  const updateNote = useUpdateNote(id);
  const deleteNote = useDeleteNote();

  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [editing, setEditing] = useState(false);

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
          feedback('Сохранено');
        },
        onError: () => feedback('Заметка изменена в другом месте. Обновите и повторите.', 'error'),
      },
    );
  };

  const remove = () => {
    deleteNote.mutate(id, {
      onSuccess: () => {
        feedback('Заметка удалена');
        navigate('/apps/notes');
      },
      onError: () => feedback('Не удалось удалить', 'error'),
    });
  };

  return (
    <Screen header={<AppHeader title="Заметка" onBack={() => navigate('/apps/notes')} />}>
      {note.isLoading ? <Skeleton className="mt-2 h-32 w-full" /> : null}

      {note.isError ? (
        <div className="pt-2">
          <Banner tone="danger" icon="lock" title="Заметка недоступна">
            Доступ мог быть отозван, а заметка удалена.
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
                placeholder="Заголовок"
                className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-strong text-ink outline-none"
              />
              <textarea
                value={body}
                onChange={(event) => setBody(event.target.value)}
                rows={10}
                className="mt-2 w-full resize-none rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none"
              />
              <div className="flex items-center justify-between pt-2">
                <button type="button" onClick={() => setEditing(false)} className="text-body-secondary text-ink-2">
                  Отмена
                </button>
                <PrimaryButton onClick={save} progress={updateNote.isPending}>
                  Сохранить
                </PrimaryButton>
              </div>
            </Card>
          ) : (
            <>
              <Card className="p-4">
                <h2 className="text-headline-md text-ink">{note.data.title || 'Без заголовка'}</h2>
                <p className="whitespace-pre-wrap pt-2 text-body-lg text-ink">{note.data.body}</p>
              </Card>

              {note.data.mine ? (
                <div className="flex gap-2 pt-3">
                  <SecondaryButton onClick={() => setEditing(true)} full>
                    Изменить
                  </SecondaryButton>
                  <SecondaryButton onClick={remove} full>
                    Удалить
                  </SecondaryButton>
                </div>
              ) : (
                <div className="px-1 pt-3 text-caption text-ink-2">
                  Заметкой поделился автор. Редактировать может только он; отзыв доступа закрывает
                  просмотр.
                </div>
              )}
            </>
          )}
        </div>
      ) : null}
    </Screen>
  );
}
