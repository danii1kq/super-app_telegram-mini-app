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
  views: number;
}

/** Query keys start with the app id so a feed refresh never resets other apps. */
export const feedKeys = {
  all: ['social', 'feed'] as const,
  comments: (postId: string) => ['social', 'comments', postId] as const,
  people: (query: string) => ['social', 'people', query] as const,
  post: (id: string) => ['social', 'post', id] as const,
  profiles: ['social', 'profile'] as const,
  profile: (id: string) => ['social', 'profile', id] as const,
  authorPostsAll: ['social', 'author-posts'] as const,
  authorPosts: (id: string) => ['social', 'author-posts', id] as const,
  viewed: (viewerId: string, postId: string) => ['social', 'viewed', viewerId, postId] as const,
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

export interface UserProfile extends Person {
  follows_you: boolean;
  is_friend: boolean;
  followers: number;
  following_count: number;
  posts_count: number;
  joined_at: string;
  telegram_url: string | null;
}

export function usePost(id: string, enabled = true) {
  return useQuery({ queryKey: feedKeys.post(id), queryFn: ({ signal }) => api<Post>(`/posts/${id}`, { signal }), enabled: Boolean(id) && enabled });
}

export function useUserProfile(id: string, enabled = true) {
  return useQuery({ queryKey: feedKeys.profile(id), queryFn: ({ signal }) => api<UserProfile>(`/people/${id}`, { signal }), enabled: Boolean(id) && enabled });
}

export function useAuthorPosts(id: string, enabled = true) {
  return useInfiniteQuery({
    queryKey: feedKeys.authorPosts(id), initialPageParam: '',
    queryFn: ({ pageParam, signal }) => api<{ posts: Post[]; next_before: string | null }>(`/people/${id}/posts?before=${pageParam}`, { signal }),
    getNextPageParam: (page) => page.next_before ?? undefined,
    select: (data) => data.pages.flatMap((page) => page.posts), enabled: Boolean(id) && enabled,
  });
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
