import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export type Difficulty = 'easy' | 'normal' | 'hard';
export interface AuraStats { str:number; agi:number; int:number; vit:number; per:number }
export interface AuraQuestion { text:string; options:number[]; topic:string; topic_name:string }
export interface DungeonEvent { id:string; name:string; rank:string; description:string; warning:string; accepted:boolean; phase:number }
export interface AuraSession {
  id:string; question_id:string; kind:'arena'|'gate'|'grind'|'tower'; question:AuraQuestion;
  round:number; rounds:number; correct:number; combo:number; best_combo:number; damage:number;
  boss_hp:number; boss_max_hp:number; boss_name:string; floor:number; seconds:number;
  hero_hp:number; hero_max_hp:number; room:number; rooms:number; phase:'combat'|'choice'|'event'; event?:DungeonEvent;
  dungeon:string; intent:string; charge:number; ability_ready:boolean;
  sets?:string[];
  asked_at:string; expires_at:string; question_deadline:string;
}
export interface AuraAbility { name:string; icon:string; text:string; cooldown_seconds:number }
export interface AuraPlayer {
  aura:number; total_aura:number; level:number; xp:number; xp_to_next:number; stat_points:number;
  stats:AuraStats; hero_class:string; class_name:string; ability:AuraAbility|null;
  farm_ready_at:string|null; arena_ready_at:string|null; gate_ready_at:string|null; tower_ready_at:string|null;
  grind_today:number; grind_cap:number; streak:number; session:AuraSession|null;
  max_hp:number; power:number; rank:string; difficulty:Difficulty; materials:Record<string,number>; clears:number; tower_floor:number;
}
export interface AuraRelic {
  key:string; name:string; rank:string; slot:string; passive:string; value:number; level:number; worn:boolean;
  description:string; effect:string; upgrade_cost:number; upgrade_materials:Record<string,number>; craft_cost:Record<string,number>; craft_aura:number;
  source:string; dungeon:string; craftable:boolean;
  set_id:string; set_name:string; set_bonus:string; lore:string; curse:string; condition_hint:string;
  favorite:boolean; next_value:number; next_description:string;
}
export interface AuraSet { id:string;name:string;description:string;keys:string[];required:number;active:boolean;equipped:number }
export interface JournalEntry { id:string;key:string;name:string;dungeon:string;hint:string;discovered:boolean;revealed:boolean }
export interface AuraClass { id:string; name:string; description:string }
export interface Dungeon { id:string; name:string; description:string; level:number; material:string; boss:string; theme:string }
export interface Quest { id:string; name:string; description:string; progress:number; goal:number; claimed:boolean; reward_aura:number; reward_materials:Record<string,number> }
export interface AuraResult { message:string; correct:boolean; answer?:number; hint?:string; aura_gain:number; xp_gain:number; damage:number; hero_damage?:number; level_up:number; relic:AuraRelic|null; bonus_relics?:AuraRelic[]; finished:boolean; won?:boolean; materials_gain?:Record<string,number>; player:AuraPlayer }
export interface LootRules { pity_after:number; dungeons:Record<string,{chance_percent:number;max_rank:string;rank_percent:Record<string,number>}>; ss_min_clears:number; sss_min_clears:number; top_rank_dungeon:string; ss_event_pity:number;sss_event_pity:number }
export interface AuraState { player:AuraPlayer; relics:AuraRelic[]; classes:AuraClass[]; catalog:AuraRelic[]; dungeons:Dungeon[]; quests:Quest[]; server_time:string; loot_rules:LootRules; event_pity:{ss:number;sss:number}; sets:AuraSet[];showcase:AuraRelic[];journal:JournalEntry[] }
export interface AuraTopEntry { user_id:string; name:string; level:number; total_aura:number; rank:number; mine:boolean; clears?:number; depth?:number; difficulty?:number; errors?:number }
export type AuraAction = 'farm'|'start'|'answer'|'class'|'stats'|'route'|'ability'|'difficulty'|'relics/equip'|'relics/upgrade'|'relics/craft'|'relics/favorite'|'quests/claim';
export const auraKeys = { state:(user:string)=>['aura','state',user] as const, top:(user:string)=>['aura','top',user] as const };
export function useAuraState(user:string) { return useQuery({queryKey:auraKeys.state(user),queryFn:()=>api<AuraState>('/aura'),enabled:!!user,refetchInterval:30000}); }
export function useAuraTop(user:string,enabled:boolean) { return useQuery({queryKey:auraKeys.top(user),queryFn:()=>api<{entries:AuraTopEntry[]}>('/aura/top'),enabled:!!user&&enabled,refetchInterval:60000}); }
export const actAura = (action:AuraAction,body:Record<string,unknown>={}) => api<AuraResult>(`/aura/${action}`,{method:'POST',body});
export const newRequestId = () => crypto.randomUUID();
