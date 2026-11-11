import { Route, Routes } from 'react-router-dom';
import { DocHomePage } from './pages/DocHomePage';

/** SruDoc space: routes mounted by the shell at /apps/doc. */
export default function DocRoutes() {
  return (
    <Routes>
      <Route index element={<DocHomePage />} />
    </Routes>
  );
}
