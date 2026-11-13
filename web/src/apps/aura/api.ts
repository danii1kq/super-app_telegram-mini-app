import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface AuraStats {
  str: number;
  agi: number;
  int: number;
  vit: number;
  per: number;
}

export interface AuraQuestion {
  text: string;
  options: number[];
  topic: string;
  topic_name: string;
}

export interface AuraSession {
  kind: 'arena' | 'gate' | 'grind' | 'tower';
  question: AuraQuestion;
  round: number;
  rounds: number;
  correct: number;
  combo: number;
  best_combo: number;
  damage: number;
  boss_hp: number;
  boss_max_hp: number;
  boss_name: string;
  floor: number;
  seconds: number;
}

export interface AuraAbility {
  name: string;
  icon: string;
  text: string;
  cooldown_seconds: number;
}

export interface AuraPlayer {
  aura: number;
  total_aura: number;
  level: number;
  xp: number;
  xp_to_next: number;
  stat_points: number;
  stats: AuraStats;
  hero_class: string;
  class_name: string;
  ability: AuraAbility | null;
  farm_ready_at: string | null;
  arena_ready_at: string | null;
  gate_ready_at: string | null;
  tower_ready_at: string | null;
  grind_today: number;
  grind_cap: number;
  streak: number;
  session: AuraSession | null;
}

export interface AuraRelic {
  key: string;
  name: string;
  rank: string;
  slot: string;
  passive: string;
  value: number;
  level: number;
  worn: boolean;
}

export interface AuraClass {
  id: string;
  name: string;
  description: string;
}

export interface AuraResult {
  message: string;
  correct: boolean;
  answer?: number;
  hint?: string;
  aura_gain: number;
  xp_gain: number;
  damage: number;
  level_up: number;
  relic: AuraRelic | null;
  finished: boolean;
  player: AuraPlayer;
}

export interface AuraState {
  player: AuraPlayer;
  relics: AuraRelic[];
  classes: AuraClass[];
  catalog: AuraRelic[];
}

export const auraKeys = {
  state: ['aura', 'state'] as const,
  top: ['aura', 'top'] as const,
};

export function useAuraState(enabled = true) {
  return useQuery({
    queryKey: auraKeys.state,
    queryFn: () => api<AuraState>('/aura'),
    enabled,
  });
}

export interface AuraTopEntry {
  user_id: string;
  name: string;
  level: number;
  total_aura: number;
  rank: number;
  mine: boolean;
}

export function useAuraTop(enabled = true) {
  return useQuery({
    queryKey: auraKeys.top,
    queryFn: () => api<{ entries: AuraTopEntry[] }>('/aura/top'),
    select: (data) => data.entries,
    enabled,
  });
}

/** One answer needs a stable id so a retry is a no-op rather than a second hit. */
export function newRequestId(): string {
  return crypto.randomUUID();
}

export function useAuraFarm() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api<AuraResult>('/aura/farm', { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auraKeys.state }),
  });
}

export function useAuraStart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (kind: string) => api<AuraResult>('/aura/start', { method: 'POST', body: { kind } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auraKeys.state }),
  });
}

export function useAuraAnswer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { choice: number; requestId: string }) =>
      api<AuraResult>('/aura/answer', {
        method: 'POST',
        body: { choice: input.choice, request_id: input.requestId },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: auraKeys.state });
      void queryClient.invalidateQueries({ queryKey: auraKeys.top });
    },
  });
}

export function useAuraClass() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: string) =>
      api<{ player: AuraPlayer }>('/aura/class', { method: 'POST', body: { class: value } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auraKeys.state }),
  });
}

export function useAuraEquip() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { key: string; slot: string }) =>
      api<{ relics: AuraRelic[] }>('/aura/relics/equip', { method: 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auraKeys.state }),
  });
}

export function useAuraStat() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (stat: string) =>
      api<{ player: AuraPlayer }>('/aura/stats', { method: 'POST', body: { stat } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auraKeys.state }),
  });
}

/** Seconds left on a cooldown, floored at zero. now is injectable so the caller
 * can drive it from a ticking clock instead of freezing at first render. */
export function secondsLeft(readyAt: string | null, now = Date.now()): number {
  if (!readyAt) return 0;
  return Math.max(0, Math.floor((new Date(readyAt).getTime() - now) / 1000));
}

/** A cooldown as "2 ч 15 мин", or an empty string when it is ready. */
export function formatCooldown(seconds: number): string {
  if (seconds <= 0) return '';
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.ceil((seconds % 3600) / 60);
  if (hours > 0) return `${hours} ч ${minutes} мин`;
  return `${minutes} мин`;
}
