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
    queryFn: () => api<{ linked: boolean; configured: boolean; import_available: boolean }>('/apps/day/status'),
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

export interface WeatherCity { id: number; name: string; region: string; country: string; latitude: number; longitude: number }
export interface WeatherDay { date: string; code: number; min: number; max: number }

export function useWeatherCities(query: string, language: string) {
  return useQuery({
    queryKey: ['day', 'weather-cities', query, language],
    queryFn: ({ signal }) => api<{ cities: WeatherCity[] }>(`/day/weather/cities?q=${encodeURIComponent(query)}&language=${language}`, { signal }),
    select: (result) => result.cities,
    enabled: query.length >= 2,
    staleTime: 15 * 60_000,
  });
}

export function useWeather(city: WeatherCity | null, timezone: string, enabled = true) {
  return useQuery({
    queryKey: ['day', 'weather', city?.latitude, city?.longitude, timezone],
    queryFn: ({ signal }) => api<{ days: WeatherDay[] }>(`/day/weather?latitude=${city?.latitude}&longitude=${city?.longitude}&timezone=${encodeURIComponent(timezone)}`, { signal }),
    select: (result) => result.days,
    enabled: city !== null && enabled,
    staleTime: 15 * 60_000,
    refetchInterval: 15 * 60_000,
  });
}
