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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: pulseKeys.feed }),
  });
}

export function useRespond() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (statusId: string) =>
      api<{ ok: boolean }>(`/pulse/${statusId}/responses`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: pulseKeys.feed }),
  });
}

export function useCloseStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (statusId: string) =>
      api<{ ok: boolean }>(`/pulse/${statusId}/close`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: pulseKeys.feed }),
  });
}
