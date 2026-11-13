import { Route, Routes } from 'react-router-dom';
import { TinderPage } from './pages/TinderPage';

/** SruTinder space: routes mounted by the shell at /apps/tinder. */
export default function TinderRoutes() {
  return (
    <Routes>
      <Route index element={<TinderPage />} />
    </Routes>
  );
}
