import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface DayEvent {
  id: string;
  title: string;
  starts_at: string;
  ends_at: string;
  all_day: boolean;
  location?: string;
  source_ref?: string;
}

export const dayKeys = {
  all: ['day'] as const,
  status: ['day', 'status'] as const,
  events: (from: string, to: string) => ['day', 'events', from, to] as const,
};

export function useDayStatus(enabled = true) {
  return useQuery({
    queryKey: dayKeys.status,
    queryFn: () => api<{ linked: boolean; configured: boolean }>('/apps/day/status'),
    enabled,
  });
}

export function useDayEvents(from: string, to: string, enabled: boolean) {
  return useQuery({
    queryKey: dayKeys.events(from, to),
    queryFn: ({ signal }) => api<{ events: DayEvent[] }>(`/apps/day/events?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`, { signal }),
    select: (data) => data.events,
    enabled,
    refetchOnWindowFocus: 'always',
  });
}
