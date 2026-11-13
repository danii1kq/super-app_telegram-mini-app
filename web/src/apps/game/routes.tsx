import { Route, Routes } from 'react-router-dom';
import { GameHomePage } from './GameHomePage';

/**
 * SruGame space: the hub, plus the games it hosts.
 *
 * The games keep their own routes under /apps/game/*, so a deep link into the
 * farm still works and the back button returns to the hub rather than the
 * catalogue.
 */
export default function GameRoutes() {
  return (
    <Routes>
      <Route index element={<GameHomePage />} />
    </Routes>
  );
}
