import { Route, Routes } from 'react-router-dom';
import { SplitHomePage } from './pages/SplitHomePage';
import { SplitGroupPage } from './pages/SplitGroupPage';

/** SruSplit space: routes mounted by the shell at /apps/split. */
export default function SplitRoutes() {
  return (
    <Routes>
      <Route index element={<SplitHomePage />} />
      <Route path=":id" element={<SplitGroupPage />} />
    </Routes>
  );
}
