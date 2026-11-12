import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface DocOrder {
  number: string;
  status: string;
  title: string;
  amount_minor: string;
  created_at: string;
  has_result: boolean;
}

export const docKeys = {
  status: ['doc', 'status'] as const,
  orders: ['doc', 'orders'] as const,
  order: (number: string) => ['doc', 'order', number] as const,
};

export function useDocStatus(enabled = true) {
  return useQuery({
    queryKey: docKeys.status,
    queryFn: () => api<{ linked: boolean }>('/apps/doc/status'),
    select: (data) => data.linked,
    enabled,
  });
}

export function useDocOrders(enabled: boolean) {
  return useQuery({
    queryKey: docKeys.orders,
    queryFn: () => api<{ orders: DocOrder[] }>('/apps/doc/orders'),
    select: (data) => data.orders,
    enabled,
  });
}

export function useDocOrder(number: string) {
  return useQuery({
    queryKey: docKeys.order(number),
    queryFn: () => api<DocOrder>(`/apps/doc/orders/${number}`),
    enabled: Boolean(number),
  });
}
