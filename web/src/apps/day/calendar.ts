export interface CalendarEvent {
  id: string;
  starts_at: string;
  ends_at: string;
  all_day: boolean;
}

export function startOfWeek(date: Date): Date {
  const start = new Date(date);
  start.setHours(0, 0, 0, 0);
  start.setDate(start.getDate() - (start.getDay() + 6) % 7);
  return start;
}

export function addDays(date: Date, count: number): Date {
  const next = new Date(date);
  next.setDate(next.getDate() + count);
  return next;
}

export function sortEvents<T extends CalendarEvent>(events: T[]): T[] {
  return [...events].sort((a, b) => Date.parse(a.starts_at) - Date.parse(b.starts_at) || a.id.localeCompare(b.id));
}

/** End times are exclusive, so an event ending at midnight does not fill the next day. */
export function eventsOnDay<T extends CalendarEvent>(events: T[], date: Date): T[] {
  const start = new Date(date);
  start.setHours(0, 0, 0, 0);
  const end = addDays(start, 1);
  return sortEvents(events.filter((event) => Date.parse(event.starts_at) < end.getTime() && Date.parse(event.ends_at) > start.getTime()));
}

export function localInput(date: Date): string {
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
}

export function parseLocalInput(value: string): Date | undefined {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) return undefined;
  const date = new Date(value);
  // Reject missing DST wall times and invalid dates instead of silently moving the meeting.
  return !Number.isNaN(date.getTime()) && localInput(date) === value ? date : undefined;
}

export function localDayKey(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}
