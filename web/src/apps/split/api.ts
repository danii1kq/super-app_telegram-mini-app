import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface SplitGroup {
  id: string;
  title: string;
  owner_id: string;
  created_at: string;
}

export interface SplitExpense {
  id: string;
  group_id: string;
  author_id: string;
  payer_id: string;
  description: string;
  amount_minor: string;
  spent_at: string;
  version: number;
  shares: Record<string, string>;
}

export interface SplitBalance {
  user_id: string;
  paid_minor: string;
  share_minor: string;
  balance_minor: string;
  settled_minor: string;
}

export const splitKeys = {
  groups: ['split', 'groups'] as const,
  expenses: (groupId: string) => ['split', 'expenses', groupId] as const,
  balances: (groupId: string) => ['split', 'balances', groupId] as const,
};

export function useSplitGroups(enabled = true) {
  return useQuery({
    queryKey: splitKeys.groups,
    queryFn: () => api<{ groups: SplitGroup[] }>('/split/groups'),
    select: (data) => data.groups,
    enabled,
  });
}

export function useSplitExpenses(groupId: string, enabled = true) {
  return useQuery({
    queryKey: splitKeys.expenses(groupId),
    queryFn: () => api<{ expenses: SplitExpense[] }>(`/split/groups/${groupId}/expenses`),
    select: (data) => data.expenses,
    enabled: Boolean(groupId) && enabled,
  });
}

export function useSplitBalances(groupId: string, enabled = true) {
  return useQuery({
    queryKey: splitKeys.balances(groupId),
    queryFn: () => api<{ balances: SplitBalance[] }>(`/split/groups/${groupId}/balances`),
    select: (data) => data.balances,
    enabled: Boolean(groupId) && enabled,
  });
}

/** Money is integer minor units; the UI shows tenge with two decimals. */
export function formatMinor(value: string): string {
  if (!/^-?\d+$/.test(value)) return '0,00';
  const minor = BigInt(value);
  const sign = minor < 0n ? '-' : '';
  const absolute = minor < 0n ? -minor : minor;
  const whole = absolute / 100n;
  const cents = String(absolute % 100n).padStart(2, '0');
  return `${sign}${whole.toLocaleString('ru-RU')},${cents}`;
}

/** Parses "1 234,56" or "1234.56" into minor units without float math. */
export function parseMinor(input: string): number | null {
  const cleaned = input.replace(/\s/g, '').replace(',', '.');
  if (!/^\d+(\.\d{1,2})?$/.test(cleaned)) return null;
  const [whole, fraction = ''] = cleaned.split('.');
  const minor = Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
  return Number.isSafeInteger(minor) && minor > 0 ? minor : null;
}
