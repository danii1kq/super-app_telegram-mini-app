import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface Note {
  id: string;
  title: string;
  body?: string;
  version: number;
  mine: boolean;
  updated_at: string;
}

export const notesKeys = {
  list: ['notes', 'list'] as const,
  one: (id: string) => ['notes', 'one', id] as const,
};

export function useNotes() {
  return useQuery({
    queryKey: notesKeys.list,
    queryFn: () => api<{ notes: Note[] }>('/notes'),
    select: (data) => data.notes,
  });
}

export function useNote(id: string) {
  return useQuery({
    queryKey: notesKeys.one(id),
    queryFn: () => api<Note>(`/notes/${id}`),
    enabled: Boolean(id),
  });
}
