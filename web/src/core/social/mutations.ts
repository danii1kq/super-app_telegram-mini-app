import { useMutation, useQueryClient, type InfiniteData } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { feedKeys, type Comment, type Post } from './api';

export function useCreatePost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { body: string; audience: Post['audience'] }) =>
      api<Post>('/posts', { method: 'POST', body: input }),
    onSuccess: () => Promise.all([
      queryClient.invalidateQueries({ queryKey: feedKeys.all }),
      queryClient.invalidateQueries({ queryKey: feedKeys.authorPostsAll }),
      queryClient.invalidateQueries({ queryKey: feedKeys.profiles }),
    ]),
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
    onSettled: (_data, _error, postId) => Promise.all([
      queryClient.invalidateQueries({ queryKey: feedKeys.all }),
      queryClient.invalidateQueries({ queryKey: feedKeys.post(postId) }),
      queryClient.invalidateQueries({ queryKey: feedKeys.authorPostsAll }),
    ]),
  });
}

export function useCreateComment(postId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => api<Comment>(`/posts/${postId}/comments`, { method: 'POST', body: { body } }),
    onSuccess: () => Promise.all([
      queryClient.invalidateQueries({ queryKey: feedKeys.comments(postId) }),
      queryClient.invalidateQueries({ queryKey: feedKeys.all }),
      queryClient.invalidateQueries({ queryKey: feedKeys.post(postId) }),
      queryClient.invalidateQueries({ queryKey: feedKeys.authorPostsAll }),
    ]),
  });
}

export function useDeletePost(onDeleted?: () => void) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) => api<{ ok: boolean }>(`/posts/${postId}`, { method: 'DELETE' }),
    onSuccess: (_data, postId) => {
      onDeleted?.();
      return Promise.all([
      queryClient.invalidateQueries({ queryKey: feedKeys.all }),
      queryClient.invalidateQueries({ queryKey: feedKeys.post(postId) }),
      queryClient.invalidateQueries({ queryKey: feedKeys.authorPostsAll }),
      queryClient.invalidateQueries({ queryKey: feedKeys.profiles }),
      ]);
    },
  });
}

export function useSetFollowing(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (following: boolean) => api<{ ok: boolean }>(`/authors/${userId}/follow`, { method: following ? 'POST' : 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['social'] }),
  });
}

export function useRecordView(postId: string, viewerId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api<{ views: number }>(`/posts/${postId}/view`, { method: 'POST' }),
    retry: 1,
    onSuccess: ({ views }) => {
      queryClient.setQueryData(feedKeys.viewed(viewerId, postId), true);
      const update = (post: Post) => post.id === postId ? { ...post, views: Math.max(post.views, views) } : post;
      queryClient.setQueryData<Post>(feedKeys.post(postId), (post) => post ? update(post) : post);
      for (const queryKey of [feedKeys.all, feedKeys.authorPostsAll]) {
        queryClient.setQueriesData<InfiniteData<{ posts: Post[]; next_before: string | null }>>({ queryKey },
          (data) => data ? { ...data, pages: data.pages.map((page) => ({ ...page, posts: page.posts.map(update) })) } : data);
      }
    },
  });
}
