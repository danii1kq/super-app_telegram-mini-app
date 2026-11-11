import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { App } from './app/App';
import { I18nProvider } from './shared/i18n';
import { ToastProvider } from './shared/ui/toast';
import { bootstrapTelegram, tg } from './shared/telegram/sdk';
import { initTheme } from './app/theme';
import './styles/index.css';

tg?.ready();
tg?.expand();

bootstrapTelegram();
initTheme();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 15_000,
    },
  },
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <ToastProvider>
          <App />
        </ToastProvider>
      </I18nProvider>
    </QueryClientProvider>
  </StrictMode>,
);
