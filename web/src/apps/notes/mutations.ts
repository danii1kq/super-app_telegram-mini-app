import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { notesKeys, type Note } from './api';

export function useCreateNote() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; body: string }) =>
      api<Note>('/notes', { method: 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: notesKeys.list }),
  });
}

export function useUpdateNote(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; body: string; version: number }) =>
      api<Note>(`/notes/${id}`, { method: 'PATCH', body: input }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notesKeys.list });
      void queryClient.invalidateQueries({ queryKey: notesKeys.one(id) });
    },
  });
}

export function useDeleteNote() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api<{ ok: boolean }>(`/notes/${id}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: notesKeys.list }),
  });
}

export function useGrantNote(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (recipients: string[]) =>
      api<{ ok: boolean }>(`/notes/${id}/grants`, {
        method: 'POST',
        body: { recipients, context_kind: 'direct' },
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: notesKeys.one(id) }),
  });
}
