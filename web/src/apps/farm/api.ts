import { api } from '@/shared/api/client';

export interface Crop { id: string; name: string; rarity: string; price: number; seed_price: number }
export interface Plot { seed: string; planted_at: string; ready_at: string; care: string; water: number; soil: number; lamp: number; genetics: number; nursery: number }
export interface Harvest { id: string; weight: number; seed: string; quality: string; mutation: string; quantity: number; value: number }
export interface Recipe { sources: string[]; crop: Crop; base: string }
export interface Breeding { source: string; result: string; started_at: string; ready_at: string }
export interface Trophy { plant: Harvest; saved_at: string }
export interface Discovery { seed: string; quality: string; mutation: string }
export interface FarmState {
	 theme: string; owned_themes: string[];
  coins: number; xp: number; seeds: Record<string, number>; plots: (Plot | null)[];
	breeding: Breeding | null; hybrids: string[]; showcase: Trophy[];
  barn: Harvest[]; collection: Record<string, Discovery>; upgrades: Record<string, number>;
  pity: Record<string, number>; opened: number; sold: number;
}
export interface Pack { id: string; name: string; price: number; weights: number[]; pity_after: number; guarantee: string }
export interface Upgrade { id: string; name: string; icon: string; description: string; benefit: string; level: number; max_level: number; cost: number }
export interface FarmTheme { id: string; name: string; description: string; stars: number }
export interface Mutation { id: string; name: string; multiplier: number; min_genetics: number; unlocked: boolean }
export interface LevelReward { level: number; title: string; coins: number; seeds: Record<string, number>; plots: number }
export interface LevelInfo { level: number; title: string; into: number; need: number; max: number; pending: number[]; rewards: LevelReward[] }
export interface FarmResponse { state: FarmState; crops: Crop[]; recipes: Recipe[]; mutations: Mutation[]; packs: Pack[]; upgrades: Upgrade[]; themes: FarmTheme[]; stars_available: boolean; server_time: string; message: string; drops: string[] | null; level: LevelInfo }
export interface FarmAction { kind: 'plant' | 'harvest' | 'sell' | 'buy' | 'pack' | 'upgrade' | 'theme' | 'redeem' | 'breed' | 'claim_breed' | 'keep' | 'return_trophy' | 'claim_level'; slot?: number; seed?: string; care?: string; pack?: string; upgrade?: string; theme?: string; code?: string; item?: string; level?: number }
export interface Leaderboard { entries: { id: string; name: string; revenue: number; rank: number; mine: boolean }[]; ends_at: string }

export const getFarm = () => api<FarmResponse>('/farm');
export const actOnFarm = (action: FarmAction) => api<FarmResponse>('/farm/actions', { method: 'POST', body: action });
export const getLeaderboard = () => api<Leaderboard>('/farm/leaderboard');
export const getThemeInvoice = (theme: string) => api<{url:string}>('/farm/themes/invoice',{method:'POST',body:{theme}});

export function countdown(milliseconds: number): string {
  const minutes = Math.max(0, Math.ceil(milliseconds / 60_000));
  return `${Math.floor(minutes / 60).toString().padStart(2, '0')}:${(minutes % 60).toString().padStart(2, '0')}`;
}
