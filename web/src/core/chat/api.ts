import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface Conversation {
  id: string;
  kind: 'direct' | 'group';
  title: string;
  last_seq: string;
  last_read: string;
  unread: number;
  updated_at: string;
}

export interface Message {
  id: string;
  conversation_id: string;
  sender_id: string;
  sender_name: string;
  seq: string;
  text: string;
  created_at: string;
}

export const chatKeys = {
  list: ['chat', 'list'] as const,
  history: (conversationId: string) => ['chat', 'history', conversationId] as const,
};

export function useConversations(enabled = true) {
  return useQuery({
    queryKey: chatKeys.list,
    queryFn: () => api<{ conversations: Conversation[] }>('/conversations'),
    select: (data) => data.conversations,
    enabled,
  });
}

export function useMessages(conversationId: string) {
  return useQuery({
    queryKey: chatKeys.history(conversationId),
    queryFn: () =>
      api<{ messages: Message[] }>(`/conversations/${conversationId}/messages?after=0&limit=100`),
    select: (data) => data.messages,
    enabled: Boolean(conversationId),
  });
}
