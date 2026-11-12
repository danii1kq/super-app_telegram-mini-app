import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useConversations, useMessages } from './api';
import { newClientMessageId, useMarkRead, useSendMessage } from './mutations';
import { MessageAttachment } from './MessageAttachment';
import { GroupMembers } from './GroupMembers';
import { ApiError } from '@/shared/api/client';
import { AppHeader, Banner, EmptyState, Icon, Screen, Skeleton } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { useI18n, useT } from '@/shared/i18n';
import { formatDayMonth, formatDateTime, formatTime } from '@/shared/i18n/date';

function nearBottom() {
  return document.documentElement.scrollHeight - window.scrollY - window.innerHeight < 120;
}

export function ChatPage() {
  const { id = '' } = useParams();
  return <ConversationView key={id} id={id} />;
}

function ConversationView({ id }: { id: string }) {
  const t = useT();
  const { lang } = useI18n();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me, status } = useSession();
  const [draft, setDraft] = useState('');
  const [membersOpen, setMembersOpen] = useState(false);
  const [attachment, setAttachment] = useState<{ file: File; kind: 'photo' | 'file' } | null>(null);
  const [attachmentPreview, setAttachmentPreview] = useState('');
  const [atBottom, setAtBottom] = useState(true);
  const [visible, setVisible] = useState(!document.hidden);
  const [composerHeight, setComposerHeight] = useState(76);
  const conversations = useConversations(status === 'ready');
  const messages = useMessages(id, status === 'ready', atBottom && visible && !membersOpen);
  const sendMessage = useSendMessage(id);
  const markRead = useMarkRead(id);
  const conversation = conversations.data?.find((item) => item.id === id);
  const composer = useRef<HTMLDivElement>(null);
  const textarea = useRef<HTMLTextAreaElement>(null);
  const photoPicker = useRef<HTMLInputElement>(null);
  const filePicker = useRef<HTMLInputElement>(null);
  const anchor = useRef<{ height: number; y: number } | null>(null);
  const stickToBottom = useRef(true);
  const previousLast = useRef('');
  const lastRead = useRef('0');
  const reading = useRef(false);
  const sending = useRef(false);
  const attempt = useRef<{ text: string; id: string; file?: File } | null>(null);
  const draftVersion = useRef(0);
  const last = messages.data?.at(-1);

  useEffect(() => {
    if (attachment?.kind !== 'photo') { setAttachmentPreview(''); return; }
    const url = URL.createObjectURL(attachment.file);
    setAttachmentPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [attachment]);

  const pickFile = (file: File | undefined, kind: 'photo' | 'file') => {
    if (!file) return;
    if (!file.size || file.size > 20 * 1024 * 1024) { feedback(t('chatFileSize'), 'error'); return; }
    setAttachment({ file, kind });
  };

  const scrollToLatest = () => {
    stickToBottom.current = true;
    window.scrollTo({ top: document.documentElement.scrollHeight });
    setAtBottom(true);
  };

  useEffect(() => {
    const onScroll = () => {
      if (anchor.current) anchor.current.y = window.scrollY;
      stickToBottom.current = nearBottom();
      setAtBottom(stickToBottom.current);
    };
    const onVisibility = () => setVisible(!document.hidden);
    window.addEventListener('scroll', onScroll, { passive: true });
    document.addEventListener('visibilitychange', onVisibility);
    return () => { window.removeEventListener('scroll', onScroll); document.removeEventListener('visibilitychange', onVisibility); };
  }, []);

  useEffect(() => {
    if (!composer.current) return;
    const observer = new ResizeObserver(([entry]) => { if (entry) setComposerHeight(entry.target.getBoundingClientRect().height); });
    observer.observe(composer.current);
    return () => observer.disconnect();
  }, []);

  useLayoutEffect(() => {
    if (anchor.current) {
      window.scrollTo({ top: anchor.current.y + document.documentElement.scrollHeight - anchor.current.height });
      anchor.current = null;
    } else if (last?.seq !== previousLast.current && stickToBottom.current) {
      scrollToLatest();
    }
    previousLast.current = last?.seq ?? '';
  }, [messages.data, last?.seq]);

  useLayoutEffect(() => { if (stickToBottom.current) scrollToLatest(); }, [composerHeight]);

  useEffect(() => {
    if (!last || !visible || !atBottom || membersOpen || !nearBottom() || reading.current || BigInt(last.seq) <= BigInt(lastRead.current)) return;
    reading.current = true;
    const seq = last.seq;
    markRead.mutate(seq, {
      onSuccess: () => { lastRead.current = seq; },
      onSettled: () => { reading.current = false; },
    });
  }, [last?.seq, visible, atBottom, membersOpen, messages.dataUpdatedAt]);

  const submit = () => {
    const text = draft.trim();
    if ((!text && !attachment) || sending.current || status !== 'ready' || !messages.data) return;
    sending.current = true;
    if (attempt.current?.text !== text || attempt.current?.file !== attachment?.file) attempt.current = { text, id: newClientMessageId(), file: attachment?.file };
    const version = draftVersion.current;
    sendMessage.mutate({ clientMessageId: attempt.current.id, text, file: attachment?.file, kind: attachment?.kind }, {
      onSuccess: () => {
        attempt.current = null;
        setAttachment(null);
        if (draftVersion.current === version) {
          setDraft('');
          if (textarea.current) textarea.current.style.height = '';
        }
        stickToBottom.current = true;
      },
      onError: (error) => feedback(error instanceof ApiError ? error.message : t('messageFailed'), 'error'),
      onSettled: () => { sending.current = false; },
    });
  };

  const loadOlder = () => {
    if (messages.isFetching) return;
    anchor.current = { height: document.documentElement.scrollHeight, y: window.scrollY };
    void messages.fetchNextPage({ throwOnError: true }).catch(() => { anchor.current = null; feedback(t('historyError'), 'error'); });
  };

  return (
    <Screen header={<AppHeader title={conversation?.title || t('chat')} onBack={() => navigate('/chats')} trailing={
      conversation?.kind === 'group' ? <button type="button" onClick={() => setMembersOpen(true)} aria-label={t('chatGroupMembers')} aria-expanded={membersOpen} className="flex size-11 items-center justify-center text-accent"><Icon name="group" size={24} /></button> : null
    } />}>
      {membersOpen && conversation?.kind === 'group' ? <GroupMembers conversationId={id} onClose={() => setMembersOpen(false)} /> : null}
      <div style={{ paddingBottom: composerHeight }} className="flex flex-col gap-2 pt-3">
        {messages.isPending ? <><Skeleton className="h-10 w-2/3" /><Skeleton className="ml-auto h-10 w-1/2" /></> : null}
        {messages.isError ? <Banner tone="danger" icon="error" title={t('historyError')}><button type="button" onClick={() => void messages.refetch()} className="min-h-11 text-accent">{t('retry')}</button></Banner> : null}
        {messages.hasNextPage ? <button type="button" disabled={messages.isFetching} onClick={loadOlder} className="min-h-11 self-center px-4 text-body-secondary text-accent disabled:opacity-50">{t(messages.isFetchingNextPage ? 'loading' : 'chatLoadOlder')}</button> : null}
        {messages.data?.length === 0 ? <EmptyState icon="forum" title={t('chatEmptyMessages')} text={t('chatStartHint')} /> : null}
        {messages.data?.map((message, index, all) => {
          const mine = message.sender_id === me?.id;
          const previous = all[index - 1];
          const showDate = !previous || new Date(previous.created_at).toDateString() !== new Date(message.created_at).toDateString();
          return (
            <div key={message.id} className="flex flex-col gap-1.5">
              {showDate ? <div className="py-2 text-center text-caption text-ink-2">{formatDayMonth(message.created_at, lang)}</div> : null}
              <div className={`max-w-[85%] min-w-20 rounded-2xl px-3 py-2 text-body-lg ${mine ? 'self-end rounded-br-md bg-accent text-btn-text' : 'self-start rounded-bl-md bg-surface text-ink'}`}>
                {!mine && conversation?.kind === 'group' ? <div className="pb-0.5 text-caption font-semibold text-accent">{message.sender_name}</div> : null}
                {message.attachment ? <MessageAttachment conversationId={id} messageId={message.id} attachment={message.attachment} /> : null}
                {message.text ? <div className="whitespace-pre-wrap break-words [overflow-wrap:anywhere]">{message.text}</div> : null}
                <div className={`flex items-center justify-end gap-1 pt-1 text-[11px] ${mine ? 'text-btn-text/75' : 'text-ink-2'}`}>
                  <time dateTime={message.created_at} title={formatDateTime(message.created_at, lang)}>{formatTime(message.created_at, lang)}</time>
                  {mine ? <span aria-label={t('messageSent')}><Icon name="check" size={14} /></span> : null}
                </div>
              </div>
            </div>
          );
        })}
      </div>
      {!atBottom ? <button type="button" onClick={scrollToLatest} className="fixed right-4 z-40 flex min-h-11 items-center gap-1 rounded-full bg-surface px-4 text-body-secondary text-accent shadow-md" style={{ bottom: composerHeight + 12 }}><Icon name="arrow_downward" size={18} />{t('chatLatest')}</button> : null}
      <div ref={composer} className="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-surface/95 backdrop-blur-xl" style={{ paddingBottom: 'var(--safe-bottom)' }}>
        {attachment ? <div className="mx-auto flex max-w-[480px] items-center gap-2 px-3 pt-2 text-ink">
          {attachmentPreview ? <img src={attachmentPreview} alt={t('chatPhoto')} className="size-12 rounded-lg object-cover" /> : <Icon name="description" size={24} />}
          <span className="min-w-0 flex-1 truncate text-caption">{attachment.file.name}</span>
          <button type="button" disabled={sendMessage.isPending} aria-label={t('chatRemoveAttachment')} onClick={() => setAttachment(null)} className="flex size-11 items-center justify-center text-ink-2 disabled:opacity-50"><Icon name="close" size={20} /></button>
        </div> : null}
        <div className="mx-auto flex w-full max-w-[480px] items-end gap-2 px-3 py-2">
          <input ref={photoPicker} type="file" accept="image/jpeg,image/png,image/webp" className="hidden" onChange={(event) => { pickFile(event.target.files?.[0], 'photo'); event.target.value = ''; }} />
          <input ref={filePicker} type="file" className="hidden" onChange={(event) => { pickFile(event.target.files?.[0], 'file'); event.target.value = ''; }} />
          <button type="button" disabled={sendMessage.isPending || status !== 'ready'} aria-label={t('chatAttachPhoto')} onClick={() => photoPicker.current?.click()} className="flex size-11 shrink-0 items-center justify-center text-accent disabled:opacity-50"><Icon name="add_photo_alternate" size={22} /></button>
          <button type="button" disabled={sendMessage.isPending || status !== 'ready'} aria-label={t('chatAttachFile')} onClick={() => filePicker.current?.click()} className="flex size-11 shrink-0 items-center justify-center text-accent disabled:opacity-50"><Icon name="attach_file" size={22} /></button>
          <textarea ref={textarea} value={draft} maxLength={4000} rows={1} aria-label={t('messagePlaceholder')} placeholder={t('messagePlaceholder')}
            onChange={(event) => { draftVersion.current++; setDraft(event.target.value); event.target.style.height = 'auto'; event.target.style.height = `${Math.min(event.target.scrollHeight, 128)}px`; }}
            onKeyDown={(event) => { if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing && event.keyCode !== 229) { event.preventDefault(); submit(); } }}
            className="max-h-32 min-h-11 min-w-0 flex-1 resize-none rounded-2xl bg-neutral-soft/70 px-3 py-2.5 text-body-lg text-ink outline-none placeholder:text-hint" />
          <button type="button" onClick={submit} disabled={(!draft.trim() && !attachment) || sendMessage.isPending || status !== 'ready' || !messages.data} aria-label={t('send')} aria-busy={sendMessage.isPending} className="flex size-11 shrink-0 items-center justify-center rounded-full bg-accent text-btn-text transition active:scale-95 disabled:opacity-50">
            <Icon name={sendMessage.isPending ? 'hourglass_top' : 'send'} size={20} filled />
          </button>
        </div>
      </div>
    </Screen>
  );
}
