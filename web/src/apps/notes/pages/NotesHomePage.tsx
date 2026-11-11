import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useNotes } from '../api';
import { useCreateNote } from '../mutations';
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
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT } from '@/shared/i18n';

/** SruNotes: own notes and notes shared with the viewer. */
export function NotesHomePage() {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const notes = useNotes();
  const createNote = useCreateNote();
  const [composing, setComposing] = useState(false);
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');

  const submit = () => {
    if (!body.trim()) return;
    createNote.mutate(
      { title: title.trim(), body: body.trim() },
      {
        onSuccess: () => {
          setTitle('');
          setBody('');
          setComposing(false);
          feedback('Заметка сохранена');
        },
        onError: () => feedback('Не удалось сохранить', 'error'),
      },
    );
  };

  return (
    <Screen header={<AppHeader title="SruNotes" onBack={() => navigate('/apps')} leadingIcon="sticky_note_2" />}>
      {composing ? (
        <Card className="mt-2 p-3">
          <input
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder="Заголовок"
            maxLength={200}
            className="w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-strong text-ink outline-none placeholder:text-hint"
          />
          <textarea
            value={body}
            onChange={(event) => setBody(event.target.value)}
            placeholder="Текст заметки"
            rows={6}
            maxLength={20000}
            className="mt-2 w-full resize-none rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <div className="flex items-center justify-between pt-2">
            <button type="button" onClick={() => setComposing(false)} className="text-body-secondary text-ink-2">
              {t('cancel')}
            </button>
            <PrimaryButton onClick={submit} disabled={!body.trim()} progress={createNote.isPending}>
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
          <Icon name="note_add" size={20} className="text-accent" />
          Новая заметка
        </button>
      )}

      <div className="pt-3">
        {notes.isLoading ? <Skeleton className="h-16 w-full" /> : null}

        {notes.isError ? (
          <Banner tone="danger" icon="error" title={t('error')}>
            Проверьте соединение и повторите.
          </Banner>
        ) : null}

        {notes.data && notes.data.length === 0 ? (
          <EmptyState
            icon="sticky_note_2"
            title={t('empty')}
            text="Заметки видны только вам, пока вы сами не поделитесь ими."
          />
        ) : null}

        {notes.data && notes.data.length > 0 ? (
          <Card>
            {notes.data.map((note, index) => (
              <div key={note.id}>
                {index > 0 ? <Hairline /> : null}
                <Row
                  icon="sticky_note_2"
                  title={note.title || 'Без заголовка'}
                  subtitle={new Date(note.updated_at).toLocaleDateString('ru-RU', {
                    day: 'numeric',
                    month: 'short',
                  })}
                  onClick={() => navigate(`/apps/notes/${note.id}`)}
                  trailing={note.mine ? undefined : <Chip tone="accent">Поделились</Chip>}
                />
              </div>
            ))}
          </Card>
        ) : null}
      </div>
    </Screen>
  );
}
