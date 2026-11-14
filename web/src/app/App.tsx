import { lazy, Suspense, useEffect, useRef } from 'react';
import { Navigate, Route, Routes, useNavigate } from 'react-router-dom';
import { useSession } from './session';
import { startParam } from '@/shared/telegram/sdk';
import { AppShell } from './AppShell';
import { ChatsPage } from '@/core/chat/ChatsPage';
import { ChatPage } from '@/core/chat/ChatPage';
import { FeedPage } from '@/core/social/FeedPage';
import { PostPage } from '@/core/social/PostPage';
import { UserProfilePage } from '@/core/profile/UserProfilePage';
import { CataloguePage } from '@/apps/CataloguePage';
import { ProfilePage } from '@/core/profile/ProfilePage';
import { Skeleton } from '@/shared/ui';

const DocApp = lazy(() => import('@/apps/doc/routes'));
const DayApp = lazy(() => import('@/apps/day/routes'));
const PulseApp = lazy(() => import('@/apps/pulse/routes'));
const NotesApp = lazy(() => import('@/apps/notes/routes'));
const SplitApp = lazy(() => import('@/apps/split/routes'));
const GameApp = lazy(() => import('@/apps/game/routes'));
const FarmApp = lazy(() => import('@/apps/farm/routes'));
const AuraApp = lazy(() => import('@/apps/aura/routes'));
const TinderApp = lazy(() => import('@/apps/tinder/routes'));

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
  const {status}=useSession();const navigate=useNavigate();const launched=useRef(false);
  useEffect(()=>{if(status!=='ready'||launched.current)return;launched.current=true;const value=startParam();if(value==='tinder')navigate('/apps/tinder',{replace:true});else if(/^split_join_[A-Za-z0-9_-]{32,64}$/.test(value))navigate(`/apps/split/join/${value.slice(11)}`,{replace:true});else if(/^split_receipt_\d+$/.test(value))navigate(`/apps/split/open/${value.slice(14)}`,{replace:true});else if(/^split_debts_\d+$/.test(value))navigate(`/apps/split/${value.slice(12)}/debts`,{replace:true});},[status,navigate]);
  return (
    <Suspense fallback={<AppFallback />}>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<Navigate to="/feed" replace />} />
          <Route path="/feed" element={<FeedPage />} />
          <Route path="/chats" element={<ChatsPage />} />
          <Route path="/chats/:id" element={<ChatPage />} />
          <Route path="/posts/:id" element={<PostPage />} />
          <Route path="/people/:id" element={<UserProfilePage />} />
          <Route path="/calendar/*" element={<DayApp />} />
          <Route path="/apps" element={<CataloguePage />} />
          <Route path="/apps/doc/*" element={<DocApp />} />
          <Route path="/apps/day/*" element={<Navigate to="/calendar" replace />} />
          <Route path="/apps/pulse/*" element={<PulseApp />} />
          <Route path="/apps/notes/*" element={<NotesApp />} />
          <Route path="/apps/split/*" element={<SplitApp />} />
          <Route path="/apps/game/*" element={<GameApp />} />
          <Route path="/apps/game/farm/*" element={<FarmApp />} />
          <Route path="/apps/game/aura/*" element={<AuraApp />} />
          <Route path="/apps/game/tinder/*" element={<TinderApp />} />
          <Route path="/apps/tinder/*" element={<TinderApp />} />
          <Route path="/apps/farm/*" element={<Navigate to="/apps/game/farm" replace />} />
          <Route path="/apps/aura/*" element={<Navigate to="/apps/game/aura" replace />} />
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="*" element={<Navigate to="/feed" replace />} />
        </Route>
      </Routes>
    </Suspense>
  );
}
