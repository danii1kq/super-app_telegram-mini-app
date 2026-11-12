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
  status: ['day', 'status'] as const,
  events: (from: string) => ['day', 'events', from] as const,
};

export function useDayStatus(enabled = true) {
  return useQuery({
    queryKey: dayKeys.status,
    queryFn: () => api<{ linked: boolean }>('/apps/day/status'),
    select: (data) => data.linked,
    enabled,
  });
}

export function useDayEvents(from: string, enabled: boolean) {
  return useQuery({
    queryKey: dayKeys.events(from),
    queryFn: () => api<{ events: DayEvent[] }>(`/apps/day/events?from=${encodeURIComponent(from)}`),
    select: (data) => data.events,
    enabled,
  });
}
