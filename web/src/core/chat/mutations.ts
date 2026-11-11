import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { chatKeys, type Message } from './api';

export function useSendMessage(conversationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { clientMessageId: string; text: string }) =>
      api<Message>(`/conversations/${conversationId}/messages`, {
        method: 'POST',
        body: { client_message_id: input.clientMessageId, text: input.text },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: chatKeys.history(conversationId) });
      void queryClient.invalidateQueries({ queryKey: chatKeys.list });
    },
  });
}

export function useMarkRead(conversationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (seq: number) =>
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
