import { Route, Routes } from 'react-router-dom';
import { PulseHomePage } from './pages/PulseHomePage';

/** SruPulse space: routes mounted by the shell at /apps/pulse. */
export default function PulseRoutes() {
  return (
    <Routes>
      <Route index element={<PulseHomePage />} />
    </Routes>
  );
}
