import { Route, Routes } from 'react-router-dom';
import { DayHomePage } from './pages/DayHomePage';

/** SruDay space: routes mounted by the shell at /apps/day. */
export default function DayRoutes() {
  return (
    <Routes>
      <Route index element={<DayHomePage />} />
    </Routes>
  );
}
