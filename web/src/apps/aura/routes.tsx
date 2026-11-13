import { Route, Routes } from 'react-router-dom';
import { AuraPage } from './pages/AuraPage';

/** SruAura space: routes mounted by the shell at /apps/game/aura. */
export default function AuraRoutes() {
  return (
    <Routes>
      <Route index element={<AuraPage />} />
    </Routes>
  );
}
