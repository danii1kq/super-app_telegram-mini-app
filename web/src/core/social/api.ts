import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
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
  comments: (postId: string) => ['social', 'comments', postId] as const,
  people: (query: string) => ['social', 'people', query] as const,
};

export function useFeed(enabled = true) {
  return useInfiniteQuery({
    queryKey: feedKeys.all,
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => api<{ posts: Post[]; next_before: string | null }>(`/feed?before=${pageParam}`, { signal }),
    getNextPageParam: (last) => last.next_before ?? undefined,
    select: (data) => data.pages.flatMap((page) => page.posts),
    enabled,
  });
}

export interface Comment {
  id: string;
  post_id: string;
  author_id: string;
  author_name: string;
  body: string;
  created_at: string;
}

export function useComments(postId: string, enabled = true) {
  return useInfiniteQuery({
    queryKey: feedKeys.comments(postId),
    staleTime: 0,
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => api<{ comments: Comment[]; next_before: string | null }>(`/posts/${postId}/comments?before=${pageParam}`, { signal }),
    getNextPageParam: (last) => last.next_before ?? undefined,
    select: (data) => data.pages.flatMap((page) => page.comments),
    enabled,
  });
}

export interface Person {
  id: string;
  public_name: string;
  username?: string;
  following: boolean;
  is_me: boolean;
}

/** The directory: everyone the viewer can write to. */
export function usePeople(enabled = true, query = '') {
  return useQuery({
    queryKey: feedKeys.people(query),
    queryFn: () => api<{ people: Person[] }>(`/people?q=${encodeURIComponent(query)}`),
    select: (data) => data.people,
    enabled,
  });
}
