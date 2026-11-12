import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

/** One purchase as the viewer experienced it. */
export interface ReportLine {
  expense_id: string;
  group_id: string;
  group_title: string;
  description: string;
  spent_at: string;
  /** The viewer's own share: what they consumed. */
  my_share: string;
  /** The whole amount, only when the viewer paid. */
  paid_by_me: string;
  category: string;
}

export interface Report {
  from: string;
  to: string;
  my_spending: string;
  paid_by_me: string;
  by_category: Record<string, string>;
  lines: ReportLine[];
  lent_out: string;
  returned_to_me: string;
  returned_by_me: string;
  owed_to_me_now: string;
  i_owe_now: string;
}

export const reportKeys = {
  month: (month: string) => ['split', 'report', month] as const,
};

/**
 * The month report.
 *
 * The server computes the month boundaries in the profile's zone, so the client
 * only names the month and never guesses a time zone.
 */
export function useReport(month: string) {
  return useQuery({
    queryKey: reportKeys.month(month),
    queryFn: () => api<Report>(`/split/report?month=${encodeURIComponent(month)}`),
    enabled: Boolean(month),
  });
}

/** The month key the report expects: 2026-11. */
export function monthKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
}

/** Shifts a month key by a number of months. */
export function shiftMonth(key: string, delta: number): string {
  const [year = 2026, month = 1] = key.split('-').map(Number);
  const date = new Date(year, month - 1 + delta, 1);
  return monthKey(date);
}

/** A human month name for the header. */
export function monthLabel(key: string, locale: string): string {
  const [year = 2026, month = 1] = key.split('-').map(Number);
  return new Date(year, month - 1, 1).toLocaleDateString(locale, {
    month: 'long',
    year: 'numeric',
  });
}
