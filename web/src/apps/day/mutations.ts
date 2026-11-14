import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, apiUpload } from '@/shared/api/client';
import { dayKeys, type DayEvent } from './api';

export function useConnectDay() {
  const cache = useQueryClient();
  return useMutation({
    mutationFn: () => api<{ linked: boolean }>('/apps/day/connect', { method: 'POST' }),
    onSuccess: () => cache.invalidateQueries({ queryKey: dayKeys.all }),
  });
}

export function useDisconnectDay() {
  const cache = useQueryClient();
  return useMutation({
    mutationFn: () => api<{ linked: boolean }>('/apps/day/connect', { method: 'DELETE' }),
    onSuccess: async () => {
      await cache.cancelQueries({ queryKey: ['day', 'events'] });
      cache.removeQueries({ queryKey: ['day', 'events'] });
      cache.setQueryData(dayKeys.status, { linked: false, configured: true });
    },
  });
}

export function useCreateDayEvent() {
  const cache = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; starts_at: string; ends_at: string; location: string; all_day?: boolean }) => api<DayEvent>('/day/events', { method: 'POST', body: input }),
    retry: false,
    onSuccess: () => cache.invalidateQueries({ queryKey: dayKeys.all }),
  });
}

export function useUpdateDayEvent() {
  const cache = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...input }: { id: string; version: number; title: string; starts_at: string; ends_at: string; location: string; all_day: boolean }) => api<DayEvent>(`/day/events/${id}`, { method: 'PATCH', body: input }),
    retry: false,
    onSuccess: () => cache.invalidateQueries({ queryKey: dayKeys.all }),
  });
}

export interface TimetableItem { title: string; starts_at: string; ends_at: string; location: string; uncertain: boolean }

export function useReadTimetable() {
  return useMutation({
    mutationFn: (input: { file: File; week: string; timezone: string }) => apiUpload<{ items: TimetableItem[] }>(`/day/import/preview?week_start=${encodeURIComponent(input.week)}&timezone=${encodeURIComponent(input.timezone)}`, input.file, input.file.type || 'application/octet-stream', 'deepseek'),
    retry: false,
  });
}
