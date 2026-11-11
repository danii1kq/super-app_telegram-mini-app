import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMessages } from './api';
import { newClientMessageId, useMarkRead, useSendMessage } from './mutations';
import { AppHeader, Banner, Icon, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';

/** Single conversation with a sticky composer. */
export function ChatPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const messages = useMessages(id);
  const sendMessage = useSendMessage(id);
  const markRead = useMarkRead(id);
  const [draft, setDraft] = useState('');
  const bottomRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'end' });
  }, [messages.data?.length]);

  useEffect(() => {
    const last = messages.data?.at(-1);
    if (!last) return;
    const seq = Number(last.seq);
    if (Number.isFinite(seq) && seq > 0) markRead.mutate(seq);
    // Marking read is a side effect of viewing the newest message.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messages.data?.length]);

  const submit = () => {
    const text = draft.trim();
    if (!text) return;
    sendMessage.mutate(
      { clientMessageId: newClientMessageId(), text },
      {
        onSuccess: () => setDraft(''),
        onError: () => feedback('Сообщение не отправлено', 'error'),
      },
    );
  };

  return (
    <Screen
      header={<AppHeader title="Чат" onBack={() => navigate('/chats')} />}
      bottomSpace
    >
      {messages.isLoading ? (
        <div className="flex flex-col gap-2 pt-2">
          <Skeleton className="h-10 w-2/3" />
          <Skeleton className="ml-auto h-10 w-1/2" />
        </div>
      ) : null}

      {messages.isError ? (
        <div className="pt-2">
          <Banner tone="danger" icon="error" title="Не удалось загрузить историю">
            Проверьте соединение и повторите.
          </Banner>
        </div>
      ) : null}

      <div className="flex flex-col gap-1.5 pt-2">
        {messages.data?.map((message) => {
          const mine = message.sender_id === me?.id;
          return (
            <div
              key={message.id}
              className={`max-w-[85%] rounded-2xl px-3 py-2 text-body-lg ${
                mine ? 'self-end bg-accent text-btn-text' : 'self-start bg-surface text-ink'
              }`}
            >
              {mine ? null : (
                <div className="pb-0.5 text-[12px] font-semibold text-accent">{message.sender_name}</div>
              )}
              <div className="whitespace-pre-wrap">{message.text}</div>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      <div
        className="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-surface/95 backdrop-blur-xl"
        style={{ paddingBottom: 'var(--safe-bottom)' }}
      >
        <div className="mx-auto flex w-full max-w-[480px] items-end gap-2 px-3 py-2">
          <textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                submit();
              }
            }}
            rows={1}
            placeholder="Сообщение"
            className="max-h-32 min-h-11 flex-1 resize-none rounded-2xl bg-neutral-soft/70 px-3 py-2.5 text-body-lg text-ink outline-none placeholder:text-hint"
          />
          <button
            type="button"
            onClick={submit}
            disabled={!draft.trim() || sendMessage.isPending}
            aria-label="Отправить"
            className="flex size-11 shrink-0 items-center justify-center rounded-full bg-accent text-btn-text transition active:scale-95 disabled:opacity-50"
          >
            <Icon name="send" size={20} filled />
          </button>
        </div>
      </div>
    </Screen>
  );
}
