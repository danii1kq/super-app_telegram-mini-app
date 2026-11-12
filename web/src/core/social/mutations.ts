import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { feedKeys, type Comment, type Post } from './api';

export function useCreatePost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { body: string; audience: Post['audience'] }) =>
      api<Post>('/posts', { method: 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: feedKeys.all }),
  });
}

export function useToggleReaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) =>
      api<{ added: boolean }>(`/posts/${postId}/reactions`, {
        method: 'POST',
        body: { kind: 'like' },
      }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: feedKeys.all }),
  });
}

export function useCreateComment(postId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => api<Comment>(`/posts/${postId}/comments`, { method: 'POST', body: { body } }),
    onSuccess: () => Promise.all([
      queryClient.invalidateQueries({ queryKey: feedKeys.comments(postId) }),
      queryClient.invalidateQueries({ queryKey: feedKeys.all }),
    ]),
  });
}

export function useDeletePost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) => api<{ ok: boolean }>(`/posts/${postId}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: feedKeys.all }),
  });
}
