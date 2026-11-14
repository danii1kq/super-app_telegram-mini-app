export function minutesLeft(expiresAt: string, now = Date.now()): number {
  return Math.max(0, Math.ceil((new Date(expiresAt).getTime() - now) / 60_000));
}

export function localDateTime(date: Date): string {
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
}

export function meetingInput(title: string, start: string, end: string) {
  const starts = new Date(start).getTime();
  const ends = new Date(end).getTime();
  if (!title.trim() || title.trim().length > 200 || !Number.isFinite(starts) || !Number.isFinite(ends)
    || ends <= starts || ends - starts > 30 * 24 * 60 * 60_000) return null;
  return { title: title.trim(), starts_at: new Date(starts).toISOString(), ends_at: new Date(ends).toISOString(), all_day: false };
}
