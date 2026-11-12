import { lazy, Suspense } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './AppShell';
import { FeedPage } from '@/core/social/FeedPage';
import { ChatsPage } from '@/core/chat/ChatsPage';
import { ChatPage } from '@/core/chat/ChatPage';
import { CreateGroupPage } from '@/core/chat/CreateGroupPage';
import { CataloguePage } from '@/apps/CataloguePage';
import { ProfilePage } from '@/core/profile/ProfilePage';
import { Skeleton } from '@/shared/ui';

const DocApp = lazy(() => import('@/apps/doc/routes'));
const DayApp = lazy(() => import('@/apps/day/routes'));
const PulseApp = lazy(() => import('@/apps/pulse/routes'));
const NotesApp = lazy(() => import('@/apps/notes/routes'));
const SplitApp = lazy(() => import('@/apps/split/routes'));

function AppFallback() {
  return (
    <div className="mx-auto w-full max-w-[480px] px-4 pt-20">
      <Skeleton className="h-6 w-40" />
      <div className="pt-4">
        <Skeleton className="h-24 w-full" />
      </div>
    </div>
  );
}

export function App() {
  return (
    <Suspense fallback={<AppFallback />}>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<Navigate to="/feed" replace />} />
          <Route path="/feed" element={<FeedPage />} />
          <Route path="/chats" element={<ChatsPage />} />
          <Route path="/chats/new-group" element={<CreateGroupPage />} />
          <Route path="/chats/:id" element={<ChatPage />} />
          <Route path="/apps" element={<CataloguePage />} />
          <Route path="/apps/doc/*" element={<DocApp />} />
          <Route path="/apps/day/*" element={<DayApp />} />
          <Route path="/apps/pulse/*" element={<PulseApp />} />
          <Route path="/apps/notes/*" element={<NotesApp />} />
          <Route path="/apps/split/*" element={<SplitApp />} />
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="*" element={<Navigate to="/feed" replace />} />
        </Route>
      </Routes>
    </Suspense>
  );
}
