import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, apiMultipart } from '@/shared/api/client';
import { chatKeys, type Conversation, type Message } from './api';

export function useSendMessage(conversationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { clientMessageId: string; text: string; file?: File; kind?: 'photo' | 'file' }) => {
      if (input.file) {
        const form = new FormData();
        form.append('client_message_id', input.clientMessageId);
        form.append('text', input.text);
        form.append('kind', input.kind ?? 'file');
        form.append('file', input.file);
        return apiMultipart<Message>(`/conversations/${conversationId}/attachments`, form);
      }
      return api<Message>(`/conversations/${conversationId}/messages`, {
        method: 'POST',
        body: { client_message_id: input.clientMessageId, text: input.text },
      });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: chatKeys.history(conversationId) });
      void queryClient.invalidateQueries({ queryKey: chatKeys.list });
    },
  });
}

export function useMarkRead(conversationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (seq: string) =>
      api<{ ok: boolean }>(`/conversations/${conversationId}/read`, {
        method: 'POST',
        body: { seq },
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: chatKeys.list }),
  });
}

/** A client id per send: a retry of the same attempt reuses it, so the server
 * returns the stored message instead of creating a duplicate. */
export function newClientMessageId(): string {
  return crypto.randomUUID();
}

/** Opens (or reuses) the one-to-one dialog with another account. */
export function useOpenDirect() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      api<{ id: string }>('/conversations', { method: 'POST', body: { user_id: userId } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: chatKeys.list }),
  });
}

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { client_group_id: string; title: string; user_ids: string[] }) =>
      api<Conversation>('/conversations/groups', { method: 'POST', body: input }),
    onSuccess: () => { void queryClient.invalidateQueries({ queryKey: chatKeys.list }); },
  });
}

export function useAddGroupMember(conversationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => api<{ ok: boolean }>(`/conversations/${conversationId}/members`, { method: 'POST', body: { user_id: userId } }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: chatKeys.members(conversationId) });
      void queryClient.invalidateQueries({ queryKey: chatKeys.list });
    },
  });
}

export function useRemoveGroupMember(conversationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => api<{ ok: boolean }>(`/conversations/${conversationId}/members/${userId}`, { method: 'DELETE' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: chatKeys.members(conversationId) });
      void queryClient.invalidateQueries({ queryKey: chatKeys.list });
    },
  });
}
