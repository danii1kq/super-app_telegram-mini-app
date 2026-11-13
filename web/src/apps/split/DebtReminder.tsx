import { useState } from 'react';
import { useSession } from '@/app/session';
import { ApiError } from '@/shared/api/client';
import { PrimaryButton, SecondaryButton, Card } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useI18n, useT } from '@/shared/i18n';
import { requestBotWriteAccess } from '@/shared/telegram/sdk';
import { useBotAccess, useReminder, useReminderAction, useSetBotAccess } from './reminders';

export function BotNotificationSettings(){
 const t=useT(),feedback=useFeedback(),access=useBotAccess(),set=useSetBotAccess();const [requesting,setRequesting]=useState(false);
 const failed=(e:unknown)=>feedback(e instanceof ApiError?e.message:t('error'),'error');
 return <Card className="mt-2 gap-3 p-3"><p className="text-body-strong text-ink">{t('splitBotNotifications')}</p>
  {!access.data?.opt_in?<SecondaryButton disabled={requesting||set.isPending} onClick={()=>{setRequesting(true);void requestBotWriteAccess().then(allowed=>{if(allowed)set.mutate({opt_in:true},{onError:failed});else feedback(t('splitBotAccessDeclined'),'error');}).finally(()=>setRequesting(false));}}>{t('splitAllowBot')}</SecondaryButton>:<label className="flex min-h-11 items-center gap-2 text-caption text-ink-2"><input className="size-5 accent-accent" type="checkbox" disabled={set.isPending} checked={access.data.enabled} onChange={e=>set.mutate({enabled:e.target.checked},{onError:failed})}/>{t('splitBotEnabled')}</label>}
  {access.data?<label className="flex min-h-11 items-center gap-2 text-caption text-ink-2"><input className="size-5 accent-accent" type="checkbox" disabled={set.isPending} checked={access.data.show_amounts} onChange={e=>set.mutate({show_amounts:e.target.checked},{onError:failed})}/>{t('splitBotShowAmounts')}</label>:null}
 </Card>;
}

export function DebtReminder({debtId}:{debtId:string}){
 const t=useT(),{lang}=useI18n(),{me}=useSession(),feedback=useFeedback(),query=useReminder(debtId),action=useReminderAction(debtId);
 const [date,setDate]=useState(()=>new Date(Date.now()+86400000).toLocaleDateString('sv-SE',{timeZone:me?.timezone||'Asia/Almaty'})),[days,setDays]=useState(7),[editing,setEditing]=useState(false);
 const agreement=query.data?.agreement;
 const run=(body:Parameters<typeof action.mutate>[0])=>action.mutate(body,{onSuccess:()=>setEditing(false),onError:e=>feedback(e instanceof ApiError?e.message:t('error'),'error')});
 return <div className="mt-3 flex flex-col gap-3 border-t border-line pt-3">
  <p className="text-caption text-ink-2">{t('splitReminderHint')}</p>
  {agreement?<><p className="text-caption text-ink">{t(agreement.status==='active'?'splitReminderActive':agreement.status==='proposed'?'splitReminderProposed':'splitReminderDisabled')}</p><p className="text-caption text-ink-2">{new Intl.DateTimeFormat(lang==='kk'?'kk-KZ':lang==='ru'?'ru-RU':'en-GB',{dateStyle:'medium',timeStyle:'short',timeZone:agreement.timezone}).format(new Date(agreement.deadline_at))} · {agreement.timezone} · {t(agreement.interval_days===1?'splitDaily':agreement.interval_days===3?'splitEveryThree':'splitWeekly')}</p>
   {agreement.status==='proposed'&&agreement.proposer_id!==me?.id?<PrimaryButton disabled={action.isPending} onClick={()=>run({version:agreement.version,action:'accept'})}>{t('splitAcceptReminder')}</PrimaryButton>:null}
   {agreement.status!=='disabled'?<SecondaryButton disabled={action.isPending} onClick={()=>run({version:agreement.version,action:'disable'})}>{t('splitDisableReminder')}</SecondaryButton>:null}
  </>:null}
  {!editing?<SecondaryButton disabled={action.isPending||query.isLoading} onClick={()=>setEditing(true)}>{t(agreement?'splitCounterReminder':'splitProposeReminder')}</SecondaryButton>:<fieldset disabled={action.isPending} className="flex flex-col gap-3">
   <label className="text-caption text-ink-2">{t('splitReminderDate')}<input aria-label={t('splitReminderDate')} type="date" className="mt-1 min-h-11 w-full rounded-lg bg-neutral-soft px-3 text-ink" value={date} onChange={e=>setDate(e.target.value)}/></label>
   <label className="text-caption text-ink-2">{t('splitReminderFrequency')}<select className="mt-1 min-h-11 w-full rounded-lg bg-neutral-soft px-3 text-ink" value={days} onChange={e=>setDays(Number(e.target.value))}><option value={1}>{t('splitDaily')}</option><option value={3}>{t('splitEveryThree')}</option><option value={7}>{t('splitWeekly')}</option></select></label>
   <PrimaryButton disabled={!date} onClick={()=>run({version:agreement?.version??0,action:'propose',date,interval_days:days})}>{t('splitProposeReminder')}</PrimaryButton><SecondaryButton onClick={()=>setEditing(false)}>{t('cancel')}</SecondaryButton>
  </fieldset>}
 </div>;
}
