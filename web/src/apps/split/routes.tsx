import { Route, Routes } from 'react-router-dom';
import { SplitHomePage } from './pages/SplitHomePage';
import { SplitGroupPage } from './pages/SplitGroupPage';
import { SplitReceiptPage } from './pages/SplitReceiptPage';
import { SplitDebtsPage } from './pages/SplitDebtsPage';
import { SplitReportPage } from './pages/SplitReportPage';

/** SruSplit space: routes mounted by the shell at /apps/split. */
export default function SplitRoutes() {
  return (
    <Routes>
      <Route index element={<SplitHomePage />} />
      <Route path="report" element={<SplitReportPage />} />
      <Route path=":groupId" element={<SplitGroupPage />} />
      <Route path=":groupId/receipt" element={<SplitReceiptPage />} />
      <Route path=":groupId/debts" element={<SplitDebtsPage />} />
    </Routes>
  );
}
