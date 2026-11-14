import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { pulseKeys, type PulseStatus } from './api';

export function useCreateStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { text: string; expiresAt: string; audience: string[] }) =>
      api<PulseStatus>('/pulse', {
        method: 'POST',
        body: { text: input.text, expires_at: input.expiresAt, audience: input.audience },
      }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: pulseKeys.feed }),
  });
}

export function useRespond() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (statusId: string) =>
      api<{ ok: boolean }>(`/pulse/${statusId}/responses`, { method: 'POST' }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: pulseKeys.feed }),
  });
}

export function useCloseStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (statusId: string) =>
      api<{ ok: boolean }>(`/pulse/${statusId}/close`, { method: 'POST' }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: pulseKeys.feed }),
  });
}

export function useCancelResponse() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (statusId: string) => api<{ ok: boolean }>(`/pulse/${statusId}/responses`, { method: 'DELETE' }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: pulseKeys.feed }),
  });
}

export function useSavePulseMeeting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; starts_at: string; ends_at: string; all_day: boolean }) =>
      api<{ id: string }>('/day/events', { method: 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['day'] }),
  });
}
