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
};

export function usePulseFeed() {
  return useQuery({
    queryKey: pulseKeys.feed,
    queryFn: () => api<{ statuses: PulseStatus[] }>('/pulse'),
    select: (data) => data.statuses,
    // Statuses expire on their own: refresh while the screen is open.
    refetchInterval: 30_000,
  });
}

/** Minutes left until the status expires, floored at zero. */
export function minutesLeft(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now();
  return Math.max(0, Math.floor(diff / 60_000));
}
