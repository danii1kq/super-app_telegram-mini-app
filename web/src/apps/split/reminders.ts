import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
export interface ReminderAgreement {debt_id:string;proposer_id:string;version:number;deadline_at:string;timezone:string;interval_days:number;status:'proposed'|'active'|'disabled';next_run_at:string}
export function useReminder(debtId:string){return useQuery({queryKey:['split','reminder',debtId],queryFn:()=>api<{agreement:ReminderAgreement|null}>(`/split/debts/${debtId}/reminder`),refetchInterval:5000});}
export function useReminderAction(debtId:string){const cache=useQueryClient();return useMutation({mutationFn:(body:{version:number;action:'propose'|'accept'|'disable';date?:string;interval_days?:number})=>api<{agreement:ReminderAgreement|null}>(`/split/debts/${debtId}/reminder`,{method:'POST',body}),onSettled:()=>{void cache.invalidateQueries({queryKey:['split','reminder',debtId]});}});}
export interface BotAccess {opt_in:boolean;enabled:boolean;show_amounts:boolean;bot_url:string}
export function useBotAccess(){return useQuery({queryKey:['split','bot-access'],queryFn:()=>api<BotAccess>('/split/bot-access')});}
export function useSetBotAccess(){const cache=useQueryClient();return useMutation({mutationFn:(body:{opt_in?:boolean;enabled?:boolean;show_amounts?:boolean})=>api<BotAccess>('/split/bot-access',{method:'POST',body}),onSuccess:data=>cache.setQueryData(['split','bot-access'],data)});}
