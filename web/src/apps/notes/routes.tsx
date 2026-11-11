import { Route, Routes } from 'react-router-dom';
import { NotesHomePage } from './pages/NotesHomePage';
import { NotePage } from './pages/NotePage';

/** SruNotes space: routes mounted by the shell at /apps/notes. */
export default function NotesRoutes() {
  return (
    <Routes>
      <Route index element={<NotesHomePage />} />
      <Route path=":id" element={<NotePage />} />
    </Routes>
  );
}
