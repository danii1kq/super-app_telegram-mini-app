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
  responses: (id: string) => ['pulse', 'responses', id] as const,
};

export function usePulseFeed(enabled = true) {
  return useQuery({
    queryKey: pulseKeys.feed,
    queryFn: () => api<{ statuses: PulseStatus[] }>('/pulse'),
    select: (data) => data.statuses,
    // Statuses expire on their own: refresh while the screen is open.
    refetchInterval: 30_000,
    refetchOnWindowFocus: 'always',
    enabled,
  });
}

export interface PulseResponder {
  id: string;
  public_name: string;
}

export function usePulseResponders(id: string, enabled: boolean) {
  return useQuery({
    queryKey: pulseKeys.responses(id),
    queryFn: ({ signal }) => api<{ people: PulseResponder[] }>(`/pulse/${id}/responses`, { signal }),
    select: (data) => data.people,
    enabled,
    staleTime: 0,
    refetchInterval: 10_000,
  });
}
