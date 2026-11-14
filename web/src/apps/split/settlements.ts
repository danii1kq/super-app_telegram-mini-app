import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';

export interface SettlementPair {debtor_id:string;creditor_id:string;transfer_minor:string;offset_minor:string;group_version:number;pending:boolean}
export interface Settlement {id:string;author_id:string;user_a:string;user_b:string;status:'proposed'|'agreed'|'reported'|'confirmed'|'rejected'|'stale';a_accepted:boolean;b_accepted:boolean;pair:SettlementPair}
export function useSettlements(groupId:string){return useQuery({queryKey:['split','settlements',groupId],queryFn:()=>api<{pairs:SettlementPair[];proposals:Settlement[]}>(`/split/groups/${groupId}/settlements`),enabled:Boolean(groupId),refetchInterval:5000,refetchOnWindowFocus:true});}
export function useSettlementAction(groupId:string){const cache=useQueryClient();return useMutation({mutationFn:({id,...body}:{id?:string;action?:string;other_id?:string;group_version?:number;client_request_id?:string})=>api<Settlement>(id?`/split/settlements/${id}/respond`:`/split/groups/${groupId}/settlements`,{method:'POST',body}),onSettled:()=>{void cache.invalidateQueries({queryKey:['split','settlements',groupId]});void cache.invalidateQueries({queryKey:['split','balances',groupId]});void cache.invalidateQueries({queryKey:['split','debts',groupId]});}});}
