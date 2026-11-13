import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface TinderProfile {
  user_id: string;
  bio: string;
  photo_url: string;
  city: string;
  age: number;
  active: boolean;
}

export interface TinderCard {
  user_id: string;
  name: string;
  username: string;
  bio: string;
  photo_url: string;
  city: string;
  age: number;
  likes_you: boolean;
  created_at: string;
}

export interface TinderMatch {
  id: string;
  partner_id: string;
  name: string;
  username: string;
  photo_url: string;
  created_at: string;
}

export const tinderKeys = {
  profile: ['tinder', 'profile'] as const,
  deck: ['tinder', 'deck'] as const,
  matches: ['tinder', 'matches'] as const,
};

export function useTinderProfile(enabled = true) {
  return useQuery({
    queryKey: tinderKeys.profile,
    queryFn: () => api<{ profile: TinderProfile | null; likes_me: number }>('/tinder/profile'),
    enabled,
  });
}

export function useTinderDeck(enabled = true) {
  return useQuery({
    queryKey: tinderKeys.deck,
    queryFn: () => api<{ cards: TinderCard[] }>('/tinder/deck'),
    select: (data) => data.cards,
    enabled,
  });
}

export function useTinderMatches(enabled = true) {
  return useQuery({
    queryKey: tinderKeys.matches,
    queryFn: () => api<{ matches: TinderMatch[] }>('/tinder/matches'),
    select: (data) => data.matches,
    enabled,
  });
}

export function useSaveProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { bio: string; photo_url: string; city: string; age: number; active: boolean }) =>
      api<{ profile: TinderProfile }>('/tinder/profile', { method: 'PUT', body: input }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: tinderKeys.profile });
      void queryClient.invalidateQueries({ queryKey: tinderKeys.deck });
    },
  });
}

export function useLike() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      api<{ matched: boolean; match_id?: string }>('/tinder/likes', {
        method: 'POST',
        body: { user_id: userId },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: tinderKeys.deck });
      void queryClient.invalidateQueries({ queryKey: tinderKeys.matches });
      void queryClient.invalidateQueries({ queryKey: tinderKeys.profile });
    },
  });
}

export function usePass() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      api<{ ok: boolean }>('/tinder/passes', { method: 'POST', body: { user_id: userId } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: tinderKeys.deck }),
  });
}
