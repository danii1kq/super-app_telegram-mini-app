import { Route, Routes } from 'react-router-dom';
import { NotesHomePage } from './pages/NotesHomePage';

/** SruNotes space: routes mounted by the shell at /apps/notes. */
export default function NotesRoutes() {
  return (
    <Routes>
      <Route index element={<NotesHomePage />} />
    </Routes>
  );
}
