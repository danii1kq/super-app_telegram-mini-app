import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { splitKeys, type SplitExpense, type SplitGroup } from './api';

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; members: string[] }) =>
      api<SplitGroup>('/split/groups', { method: 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: splitKeys.groups }),
  });
}

export function useAddExpense(groupId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      clientRequestId: string;
      description: string;
      amountMinor: number;
      payerId: string;
      participants: string[];
    }) =>
      api<SplitExpense>(`/split/groups/${groupId}/expenses`, {
        method: 'POST',
        body: {
          client_request_id: input.clientRequestId,
          description: input.description,
          amount_minor: String(input.amountMinor),
          payer_id: input.payerId,
          participants: input.participants,
        },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: splitKeys.expenses(groupId) });
      void queryClient.invalidateQueries({ queryKey: splitKeys.balances(groupId) });
    },
  });
}

/** A request id per add: a retry of the same attempt reuses it, so the server
 * returns the stored expense instead of creating a duplicate. */
export function newRequestId(): string {
  return crypto.randomUUID();
}
