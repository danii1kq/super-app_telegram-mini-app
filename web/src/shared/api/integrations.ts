import { useMutation } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface DeadlineResult {
  event_id: string;
  event_title: string;
  order_number: string;
  due_date: string;
}

/** SruDoc -> SruDay: create a calendar deadline for an order. */
export function useDocDeadline() {
  return useMutation({
    mutationFn: (input: {
      idempotencyKey: string;
      orderNumber: string;
      title: string;
      dueDate: string;
    }) =>
      api<DeadlineResult>('/integrations/doc-deadline', {
        method: 'POST',
        body: {
          idempotency_key: input.idempotencyKey,
          order_number: input.orderNumber,
          title: input.title,
          due_date: input.dueDate,
        },
      }),
  });
}

/** SruNotes -> chat: grant read access and get the card to post. */
export function useShareNote() {
  return useMutation({
    mutationFn: (input: { noteId: string; recipients: string[] }) =>
      api<{ kind: string; app: string; id: string; title: string }>('/integrations/note-share', {
        method: 'POST',
        body: { note_id: input.noteId, recipients: input.recipients },
      }),
  });
}

export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}
