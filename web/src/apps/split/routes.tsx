import { Route, Routes } from 'react-router-dom';
import { SplitHomePage } from './pages/SplitHomePage';

/** SruSplit space: routes mounted by the shell at /apps/split. */
export default function SplitRoutes() {
  return (
    <Routes>
      <Route index element={<SplitHomePage />} />
    </Routes>
  );
}
