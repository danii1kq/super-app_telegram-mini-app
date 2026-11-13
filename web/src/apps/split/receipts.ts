import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, apiUpload } from '@/shared/api/client';

/** One line of a receipt draft. */
export interface ReceiptItem {
  id: string;
  position: number;
  name: string;
  quantity: number;
  amount_minor: string;
  unclear: boolean;
  /** How many units of this line each person takes, keyed by user id. */
  units: Record<string, number>;
  allocation_mode: 'units' | 'shared';
  parent_position: number | null;
  suggested_parent_position?: number | null;
}

export interface Receipt {
  id: string;
  group_id: string;
  author_id: string;
  status: 'draft' | 'collecting' | 'review' | 'saved' | 'cancelled';
  version: number;
  total_minor: string;
  discount_minor: string;
  service_minor: string;
  items: ReceiptItem[];
  created_at: string;
  /** Stable code, translated by the UI: unavailable or unreadable. */
  ocr_error?: 'unavailable' | 'unreadable';
  saved_expense_id?: string;
  image_state?: 'none' | 'stored' | 'purged' | 'processing';
  image_bytes?: number;
  image_width?: number;
  image_height?: number;
  payer_id?: string;
  description?: string;
  category?: string;
  approvals: { user_id: string; decision: 'pending' | 'accepted' | 'question'; question: string }[];
}

export interface Allocation {
  total_minor: string;
  discount_minor: string;
  service_minor: string;
  per_person: Record<string, string>;
  owes_to_payer: Record<string, string>;
  sum_shares: string;
  difference: string;
  unallocated: string[];
}

export interface DebtPayment {
  status: 'pending' | 'confirmed' | 'rejected';
  method: 'transfer' | 'offset';
  id: string;
  amount_minor: string;
  paid_at: string;
  author_id: string;
}

export interface Debt {
  version: number;
  id: string;
  group_id: string;
  author_id: string;
  creditor_id: string;
  debtor_id: string;
  amount_minor: string;
  paid_minor: string;
  outstanding_minor: string;
  note: string;
  payments: DebtPayment[];
  created_at: string;
  /** Set when the debt points at an expense that already carries the money. */
  source_expense_id?: string;
}

export const receiptKeys = {
  one: (id: string) => ['split', 'receipt', id] as const,
  allocation: (id: string, payerId: string) => ['split', 'receipt', id, 'allocation', payerId] as const,
};

export const debtKeys = {
  list: (groupId: string) => ['split', 'debts', groupId] as const,
  members: (groupId: string) => ['split', 'members', groupId] as const,
};

/**
 * The group's own members.
 *
 * A receipt or an expense may only involve people of this group, so pickers use
 * this list and never the app-wide directory.
 */
export function useGroupMembers(groupId: string) {
  return useQuery({
    queryKey: debtKeys.members(groupId),
    queryFn: () => api<{ members: string[] }>(`/split/groups/${groupId}/members`),
    select: (data) => data.members,
    enabled: Boolean(groupId),
  });
}

export function useGroupPeople(groupId: string) {
  return useQuery({queryKey:debtKeys.members(groupId),queryFn:()=>api<{members:string[];people:{id:string;public_name:string}[]}>(`/split/groups/${groupId}/members`),select:r=>r.people,enabled:Boolean(groupId),refetchInterval:5000});
}

export function useReceipt(id: string) {
  return useQuery({
    queryKey: receiptKeys.one(id),
    queryFn: () => api<Receipt>(`/split/receipts/${id}`),
    enabled: Boolean(id),
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  });
}

export function useReceipts(groupId: string) {
  return useQuery({ queryKey: ['split', 'receipts', groupId], queryFn: () => api<{receipts: Receipt[]}>(`/split/groups/${groupId}/receipts`), select: r => r.receipts, enabled: Boolean(groupId), refetchInterval: 5000 });
}

export function useReceiptAction(id: string, groupId: string) {
  const cache=useQueryClient();
  return useMutation({ mutationFn: ({action,...body}: {action:string;version?:number;item_id?:string;count?:number;decision?:string;question?:string;payer_id?:string;client_request_id?:string;description?:string;category?:string}) => api<Receipt>(`/split/receipts/${id}/${action}`,{method:'POST',body}),
   onSuccess:r=>{cache.setQueryData(receiptKeys.one(id),r);void cache.invalidateQueries({queryKey:['split','receipts',groupId]});void cache.invalidateQueries({queryKey:['split','expenses',groupId]});void cache.invalidateQueries({queryKey:['split','balances',groupId]});void cache.invalidateQueries({queryKey:debtKeys.list(groupId)});},
   onError:()=>{void cache.invalidateQueries({queryKey:receiptKeys.one(id)});}
  });
}

export function useAllocation(id: string, payerId: string) {
  return useQuery({
    queryKey: receiptKeys.allocation(id, payerId),
    queryFn: () => api<Allocation>(`/split/receipts/${id}/allocation?payer_id=${payerId}`),
    enabled: Boolean(id) && Boolean(payerId),
  });
}

export function useDebts(groupId: string) {
  return useQuery({
    queryKey: debtKeys.list(groupId),
    queryFn: () => api<{ debts: Debt[] }>(`/split/groups/${groupId}/debts`),
    select: (data) => data.debts,
    enabled: Boolean(groupId),
    refetchInterval: 5000,
    refetchOnWindowFocus: true,
  });
}

export function useRespondPayment(groupId: string) {
 const cache=useQueryClient();
 return useMutation({mutationFn:({debtId,paymentId,accept}:{debtId:string;paymentId:string;accept:boolean})=>api<Debt>(`/split/debts/${debtId}/payments/${paymentId}/respond`,{method:'POST',body:{accept}}),onSuccess:()=>{void cache.invalidateQueries({queryKey:debtKeys.list(groupId)});void cache.invalidateQueries({queryKey:['split','balances',groupId]});}});
}

/** Uploads the photo as a raw body: the server sniffs the real format. */
export function useUploadReceipt(groupId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) =>
      apiUpload<Receipt>(
        `/split/groups/${groupId}/receipts`,
        file,
        file.type || 'application/octet-stream',
        'deepseek',
      ),
    onSuccess: (receipt) => {
      queryClient.setQueryData(receiptKeys.one(receipt.id), receipt);
    },
  });
}

/** Starts an empty draft for a receipt typed by hand. */
export function useManualReceipt(groupId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api<Receipt>(`/split/groups/${groupId}/receipts/manual`, { method: 'POST' }),
    onSuccess: (receipt) => {
      queryClient.setQueryData(receiptKeys.one(receipt.id), receipt);
    },
  });
}

export function useUpdateReceipt(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      version: number;
      totalMinor: string;
      discountMinor: string;
      serviceMinor: string;
      items: { id?: string; name: string; quantity: number; amountMinor: string; unclear: boolean; units: Record<string, number>; allocationMode?: 'units' | 'shared'; parentPosition?: number | null }[];
    }) =>
      api<Receipt>(`/split/receipts/${id}`, {
        method: 'PATCH',
        body: {
          version: input.version,
          total_minor: input.totalMinor,
          discount_minor: input.discountMinor,
          service_minor: input.serviceMinor,
          items: input.items.map((item) => ({
            id: item.id,
            allocation_mode: item.allocationMode,
            parent_position: item.parentPosition,
            name: item.name,
            quantity: item.quantity,
            amount_minor: item.amountMinor,
            unclear: item.unclear,
            units: item.units,
          })),
        },
      }),
    onSuccess: (receipt) => {
      queryClient.setQueryData(receiptKeys.one(id), receipt);
      void queryClient.invalidateQueries({ queryKey: ['split', 'receipt', id] });
    },
  });
}

export function useSaveReceipt(id: string, groupId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { payerId: string; clientRequestId: string; description: string; category?: string }) =>
      api<{ id: string }>(`/split/receipts/${id}/save`, {
        method: 'POST',
        body: {
          payer_id: input.payerId,
          client_request_id: input.clientRequestId,
          description: input.description,
          category: input.category,
        },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['split', 'expenses', groupId] });
      void queryClient.invalidateQueries({ queryKey: ['split', 'balances', groupId] });
      void queryClient.invalidateQueries({ queryKey: debtKeys.list(groupId) });
    },
  });
}

export function useCreateDebt(groupId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      clientRequestId: string;
      creditorId: string;
      debtorId: string;
      amountMinor: string;
      note: string;
    }) =>
      api<Debt>(`/split/groups/${groupId}/debts`, {
        method: 'POST',
        body: {
          client_request_id: input.clientRequestId,
          creditor_id: input.creditorId,
          debtor_id: input.debtorId,
          amount_minor: input.amountMinor,
          note: input.note,
        },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: debtKeys.list(groupId) });
      void queryClient.invalidateQueries({ queryKey: ['split', 'balances', groupId] });
    },
  });
}

/** Records that an expense share is tracked as a debt. Never adds the amount. */
export function useRecordExpenseDebt(groupId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { expenseId: string; debtorId: string; clientRequestId: string }) =>
      api<Debt>(`/split/groups/${groupId}/expenses/${input.expenseId}/debt`, {
        method: 'POST',
        body: { debtor_id: input.debtorId, client_request_id: input.clientRequestId },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: debtKeys.list(groupId) });
      void queryClient.invalidateQueries({ queryKey: ['split', 'balances', groupId] });
    },
  });
}

export function useAddPayment(groupId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { debtId: string; amountMinor: string; clientRequestId: string }) =>
      api<Debt>(`/split/debts/${input.debtId}/payments`, {
        method: 'POST',
        body: { amount_minor: input.amountMinor, client_request_id: input.clientRequestId },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: debtKeys.list(groupId) });
      void queryClient.invalidateQueries({ queryKey: ['split', 'balances', groupId] });
    },
  });
}
