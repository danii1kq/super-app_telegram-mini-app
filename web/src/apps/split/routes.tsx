import { Navigate, Route, Routes, useParams } from 'react-router-dom';
import { useReceipt } from './receipts';
import { SplitHomePage } from './pages/SplitHomePage';
import { SplitGroupPage } from './pages/SplitGroupPage';
import { SplitReceiptPage } from './pages/SplitReceiptPage';
import { SplitDebtsPage } from './pages/SplitDebtsPage';
import { SplitReportPage } from './pages/SplitReportPage';
import { ReceiptDetailsPage } from './pages/ReceiptDetailsPage';
import { SplitInvitePage } from './pages/SplitInvitePage';
import { SplitSettlementsPage } from './pages/SplitSettlementsPage';

/** SruSplit space: routes mounted by the shell at /apps/split. */
export default function SplitRoutes() {
  return (
    <Routes>
      <Route index element={<SplitHomePage />} />
      <Route path="report" element={<SplitReportPage />} />
      <Route path="join/:token" element={<SplitInvitePage />} />
      <Route path="open/:receiptId" element={<OpenReceipt />} />
      <Route path=":groupId" element={<SplitGroupPage />} />
      <Route path=":groupId/receipt" element={<SplitReceiptPage />} />
      <Route path=":groupId/receipts/:receiptId" element={<ReceiptDetailsPage />} />
      <Route path=":groupId/debts" element={<SplitDebtsPage />} />
      <Route path=":groupId/settlements" element={<SplitSettlementsPage />} />
    </Routes>
  );
}

function OpenReceipt(){const {receiptId=''}=useParams();const receipt=useReceipt(receiptId);return receipt.data?<Navigate replace to={`/apps/split/${receipt.data.group_id}/receipts/${receiptId}`}/>:receipt.isError?<Navigate replace to="/apps/split"/>:null;}
