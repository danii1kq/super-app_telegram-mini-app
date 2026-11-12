import { api } from '@/shared/api/client';

export interface Crop { id: string; name: string; rarity: string; price: number; seed_price: number }
export interface Plot { seed: string; planted_at: string; ready_at: string; care: string; water: number; soil: number; lamp: number; genetics: number; nursery: number }
export interface Harvest { id: string; weight: number; seed: string; quality: string; mutation: string; quantity: number; value: number }
export interface Recipe { sources: string[]; crop: Crop; base: string }
export interface Breeding { source: string; result: string; started_at: string; ready_at: string }
export interface Trophy { plant: Harvest; saved_at: string }
export interface Discovery { seed: string; quality: string; mutation: string }
export interface FarmState {
	crash?: CrashRound; game_history: GameResult[];
	 theme: string; owned_themes: string[];
  coins: number; xp: number; seeds: Record<string, number>; plots: (Plot | null)[];
	breeding: Breeding | null; hybrids: string[]; showcase: Trophy[];
	expedition: Expedition | null;
  barn: Harvest[]; collection: Record<string, Discovery>; upgrades: Record<string, number>;
  pity: Record<string, number>; opened: number; sold: number;
  claimed_levels: number[]; orders: OrderBoard;
}
export interface Pack { id: string; name: string; price: number; weights: number[]; pity_after: number; guarantee: string }
export interface Upgrade { id: string; name: string; icon: string; description: string; benefit: string; level: number; max_level: number; cost: number }
export interface FarmTheme { id: string; name: string; description: string; stars: number }
export interface Mutation { id: string; name: string; multiplier: number; min_genetics: number; unlocked: boolean }
export interface Order { id: string; seed: string; quality: string; mutation: string; quantity: number; coins: number; xp: number }
export interface OrderBoard { day: string; orders: Order[]; fulfilled: string[] }
export interface DailyReward { day: number; coins: number; seeds: Record<string, number>; xp: number }
export interface DailyInfo { day: number; streak: number; best: number; claimable: boolean; rewards: DailyReward[] }
export interface LevelReward { level: number; title: string; coins: number; seeds: Record<string, number>; plots: number }
export interface LevelInfo { level: number; title: string; into: number; need: number; max: number; pending: number[]; rewards: LevelReward[] }
export interface ExpeditionRoute { id: string; name: string; description: string; price: number; duration_seconds: number; seeds: Record<string,number>; xp: number }
export interface Expedition { route: string; name: string; started_at: string; ready_at: string; seeds: Record<string,number>; xp: number }
export interface CrashRound { id: string; stake: number; started_at: string; status: 'running'|'cashed'|'busted'; multiplier_bp: number; payout: number; bust_bp?: number; return_percent: number }
export interface GameResult { id: string; kind: 'crash'|'flip'|'upgrade'; status: string; stake: number; payout: number; multiplier_bp?: number; seed?: string; mutation?: string; chance_bp?: number; side?: string; at: string }
export interface GameRules { min_stake: number; max_stake: number; crash_max_bp: number; crash_return_percent: number; crash_growth: number; flip_fee_percent: number; flip_expiry_seconds: number; upgrade_return_percent: number; upgrade_max_chance_bp: number }
export interface Coinflip { id: string; host_id: string; host_name: string; guest_id: string; guest_name: string; stake: number; host_side: string; result_side: string; winner_id: string; status: string; created_at: string; expires_at: string; fee_percent: number }
export interface UpgradeQuote { source_value: number; target_value: number; chance_bp: number; stake: number; target: Harvest }
export interface FarmResponse { state: FarmState; crops: Crop[]; recipes: Recipe[]; mutations: Mutation[]; packs: Pack[]; upgrades: Upgrade[]; themes: FarmTheme[]; stars_available: boolean; server_time: string; message: string; drops: string[] | null; level: LevelInfo; daily: DailyInfo; expeditions: ExpeditionRoute[]; games: GameRules }
export interface FarmAction { kind: 'plant' | 'harvest' | 'sell' | 'buy' | 'pack' | 'upgrade' | 'theme' | 'redeem' | 'breed' | 'claim_breed' | 'keep' | 'return_trophy' | 'claim_level' | 'fulfill_order' | 'claim_daily' | 'start_expedition' | 'claim_expedition' | 'crash_start' | 'crash_cashout' | 'plant_upgrade' | 'flip_create' | 'flip_join' | 'flip_cancel'; slot?: number; seed?: string; care?: string; pack?: string; upgrade?: string; theme?: string; code?: string; item?: string; level?: number; stake?: number; side?: string; request_id?: string }
export interface Leaderboard { entries: { id: string; name: string; revenue: number; rank: number; mine: boolean }[]; ends_at: string }

export const getFarm = () => api<FarmResponse>('/farm');
export const actOnFarm = (action: FarmAction) => api<FarmResponse>('/farm/actions', { method: 'POST', body: action });
export const getLeaderboard = () => api<Leaderboard>('/farm/leaderboard');
export const getThemeInvoice = (theme: string) => api<{url:string}>('/farm/themes/invoice',{method:'POST',body:{theme}});
export const getCoinflips = () => api<{flips:Coinflip[];server_time:string}>('/farm/coinflips');
export const getUpgradeQuote = (body: Pick<FarmAction,'item'|'seed'|'upgrade'|'stake'>) => api<UpgradeQuote>('/farm/upgrade-quote',{method:'POST',body});

export function countdown(milliseconds: number): string {
  const minutes = Math.max(0, Math.ceil(milliseconds / 60_000));
  return `${Math.floor(minutes / 60).toString().padStart(2, '0')}:${(minutes % 60).toString().padStart(2, '0')}`;
}
