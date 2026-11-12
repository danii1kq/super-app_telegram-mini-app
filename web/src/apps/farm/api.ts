import { api } from '@/shared/api/client';

export interface Crop { id: string; name: string; rarity: string; price: number; seed_price: number }
export interface Plot { seed: string; planted_at: string; ready_at: string; care: string; water: number; soil: number; lamp: number }
export interface Harvest { seed: string; quality: string; mutation: string; quantity: number; value: number }
export interface Discovery { seed: string; quality: string; mutation: string }
export interface FarmState {
  coins: number; xp: number; seeds: Record<string, number>; plots: (Plot | null)[];
  barn: Harvest[]; collection: Record<string, Discovery>; upgrades: Record<string, number>;
  pity: Record<string, number>; opened: number; sold: number;
}
export interface Pack { id: string; name: string; price: number; weights: number[] }
export interface FarmResponse { state: FarmState; crops: Crop[]; packs: Pack[]; server_time: string; message: string; drops: string[] | null }
export interface FarmAction { kind: 'plant' | 'harvest' | 'sell' | 'buy' | 'pack' | 'upgrade'; slot?: number; seed?: string; care?: string; pack?: string; upgrade?: string }
export interface Leaderboard { entries: { id: string; name: string; revenue: number; rank: number; mine: boolean }[]; ends_at: string }

export const getFarm = () => api<FarmResponse>('/farm');
export const actOnFarm = (action: FarmAction) => api<FarmResponse>('/farm/actions', { method: 'POST', body: action });
export const getLeaderboard = () => api<Leaderboard>('/farm/leaderboard');

export function countdown(milliseconds: number): string {
  const minutes = Math.max(0, Math.ceil(milliseconds / 60_000));
  return `${Math.floor(minutes / 60).toString().padStart(2, '0')}:${(minutes % 60).toString().padStart(2, '0')}`;
}
