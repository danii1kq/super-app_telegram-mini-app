import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface Post {
  id: string;
  author_id: string;
  author_name: string;
  body: string;
  audience: 'public' | 'followers' | 'community';
  community_id?: string;
  created_at: string;
  likes: number;
  comments: number;
  liked_by_me: boolean;
}

/** Query keys start with the app id so a feed refresh never resets other apps. */
export const feedKeys = {
  all: ['social', 'feed'] as const,
};

export function useFeed(enabled = true) {
  return useQuery({
    queryKey: feedKeys.all,
    queryFn: () => api<{ posts: Post[] }>('/feed'),
    select: (data) => data.posts,
    enabled,
  });
}
