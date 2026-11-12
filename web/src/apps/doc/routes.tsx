import { Route, Routes } from 'react-router-dom';
import { DocHomePage } from './pages/DocHomePage';
import { DocOrderPage } from './pages/DocOrderPage';

/** SruDoc space: routes mounted by the shell at /apps/doc. */
export default function DocRoutes() {
  return (
    <Routes>
      <Route index element={<DocHomePage />} />
      <Route path="order/:number" element={<DocOrderPage />} />
    </Routes>
  );
}
