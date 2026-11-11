import { useNavigate } from 'react-router-dom';
import { AppHeader, Screen } from '@/shared/ui';
import { Placeholder } from '@/shared/ui/Placeholder';

/** SruNotes entry screen. */
export function NotesHomePage() {
  const navigate = useNavigate();
  return (
    <Screen header={<AppHeader title="SruNotes" onBack={() => navigate('/apps')} leadingIcon="sticky_note_2" />}>
      <Placeholder title="SruNotes" icon="sticky_note_2" text="Заметки появятся после подключения API." />
    </Screen>
  );
}
