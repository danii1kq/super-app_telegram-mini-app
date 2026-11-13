import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface PulseStatus {
  id: string;
  author_id: string;
  author_name: string;
  text: string;
  expires_at: string;
  created_at: string;
  closed_at?: string;
  responses: number;
  mine: boolean;
  responded: boolean;
}

export const pulseKeys = {
  feed: ['pulse', 'feed'] as const,
  responders: (statusId: string) => ['pulse', 'responders', statusId] as const,
};

export interface PulseResponder {
  user_id: string;
  name: string;
  username: string;
  answered_at: string;
}

/** Who answered a status. Only the author may read this list. */
export function usePulseResponders(statusId: string, enabled = true) {
  return useQuery({
    queryKey: pulseKeys.responders(statusId),
    queryFn: () => api<{ responders: PulseResponder[] }>(`/pulse/${statusId}/responses`),
    select: (data) => data.responders,
    enabled: Boolean(statusId) && enabled,
  });
}

export function usePulseFeed(enabled = true) {
  return useQuery({
    queryKey: pulseKeys.feed,
    queryFn: () => api<{ statuses: PulseStatus[] }>('/pulse'),
    select: (data) => data.statuses,
    // Statuses expire on their own: refresh while the screen is open.
    refetchInterval: 30_000,
    enabled,
  });
}

/** Minutes left until the status expires, floored at zero. */
export function minutesLeft(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now();
  return Math.max(0, Math.floor(diff / 60_000));
}
