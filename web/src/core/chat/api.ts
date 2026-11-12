import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { orderedMessages } from './history';

export interface Conversation {
  id: string;
  kind: 'direct' | 'group';
  title: string;
  last_seq: string;
  last_read: string;
  unread: number;
  updated_at: string;
  partner_id: string | null;
  last_message: Pick<Message, 'text' | 'sender_id' | 'sender_name' | 'created_at' | 'attachment'> | null;
}

export interface Attachment {
  kind: 'photo' | 'file';
  name: string;
  mime: string;
  size: number;
  width?: number;
  height?: number;
}

export interface Message {
  id: string;
  conversation_id: string;
  sender_id: string;
  sender_name: string;
  seq: string;
  text: string;
  created_at: string;
  attachment: Attachment | null;
}

export interface GroupMember {
  user_id: string;
  public_name: string;
  role: 'owner' | 'member';
}

export const chatKeys = {
  list: ['chat', 'list'] as const,
  history: (conversationId: string) => ['chat', 'history', conversationId] as const,
  members: (conversationId: string) => ['chat', 'members', conversationId] as const,
};

export function useGroupMembers(conversationId: string, enabled = true) {
  return useQuery({
    queryKey: chatKeys.members(conversationId),
    queryFn: ({ signal }) => api<{ members: GroupMember[] }>(`/conversations/${conversationId}/members`, { signal }),
    select: (data) => data.members,
    enabled: Boolean(conversationId) && enabled,
    staleTime: 0,
    refetchOnWindowFocus: 'always',
    refetchInterval: 10_000,
  });
}

export function useConversations(enabled = true) {
  return useQuery({
    queryKey: chatKeys.list,
    queryFn: () => api<{ conversations: Conversation[] }>('/conversations'),
    select: (data) => data.conversations,
    enabled,
    refetchOnWindowFocus: 'always',
    refetchInterval: 10_000,
  });
}

export function useMessages(conversationId: string, enabled = true, live = true) {
  return useInfiniteQuery({
    queryKey: chatKeys.history(conversationId),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      api<{ messages: Message[]; has_more: boolean }>(
        `/conversations/${conversationId}/messages?limit=50${pageParam ? `&before=${pageParam}` : ''}`,
        { signal },
      ),
    getNextPageParam: (page) => page.has_more ? page.messages[0]?.seq : undefined,
    select: (data) => orderedMessages(data.pages),
    enabled: Boolean(conversationId) && enabled,
    refetchOnWindowFocus: live ? 'always' : false,
    refetchOnReconnect: live ? 'always' : false,
    // shortcut: loaded pages refresh together; use targeted SSE updates if long histories make this costly.
    refetchInterval: live ? 5_000 : false,
  });
}
