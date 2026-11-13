import { useNavigate, useParams } from 'react-router-dom';
import { useSession } from '@/app/session';
import { AppHeader, Banner, Card, PrimaryButton, SecondaryButton, Screen } from '@/shared/ui';
import { ApiError } from '@/shared/api/client';
import { useT, type DictKey } from '@/shared/i18n';
import { useFeedback } from '@/shared/ui/toast';
import { formatMinor } from '../api';
import { useGroupPeople } from '../receipts';
import { newRequestId } from '../mutations';
import { useSettlementAction, useSettlements, type Settlement, type SettlementPair } from '../settlements';

const status:Record<Settlement['status'],DictKey>={proposed:'splitSettlementProposed',agreed:'splitSettlementAgreed',reported:'splitPaymentAwaiting',confirmed:'splitPaymentConfirmed',rejected:'splitPaymentRejected',stale:'splitSettlementStale'};
export function SplitSettlementsPage(){
 const {groupId=''}=useParams(),navigate=useNavigate(),t=useT(),feedback=useFeedback(),{me}=useSession();
 const plan=useSettlements(groupId),action=useSettlementAction(groupId),people=useGroupPeople(groupId);
 const name=(id:string)=>id===me?.id?t('you'):people.data?.find(p=>p.id===id)?.public_name??`${t('member')} ${id}`;
 const related=(p:SettlementPair)=>p.debtor_id===me?.id||p.creditor_id===me?.id;
 const text=(p:SettlementPair)=><><p className="text-body-strong text-ink">{name(p.debtor_id)} → {name(p.creditor_id)}: {formatMinor(p.transfer_minor)} ₸</p><p className="text-caption text-ink-2">{t('splitOffset')}: {formatMinor(p.offset_minor)} ₸</p></>;
 const run=(body:Parameters<typeof action.mutate>[0])=>action.mutate(body,{onError:e=>feedback(e instanceof ApiError?e.message:t('error'),'error')});
 return <Screen header={<AppHeader title={t('splitHowToSettle')} onBack={()=>navigate(`/apps/split/${groupId}`)}/>}>
  <p className="py-3 text-caption text-ink-2">{t('splitSettlementHint')}</p>
  {plan.isError?<Banner tone="danger" icon="error" title={t('error')}>{t('retryConnection')}</Banner>:null}
  {plan.data?.pairs.filter(related).map(p=>{const active=plan.data.proposals.find(v=>['proposed','agreed','reported'].includes(v.status)&&[v.user_a,v.user_b].includes(p.debtor_id)&&[v.user_a,v.user_b].includes(p.creditor_id));return <Card key={`${p.debtor_id}:${p.creditor_id}`} className="mb-2 gap-3 p-4">{text(p)}{p.pending?<p className="text-caption text-warning">{t('splitPaymentAwaiting')}</p>:!active?<PrimaryButton disabled={action.isPending} onClick={()=>run({other_id:p.debtor_id===me?.id?p.creditor_id:p.debtor_id,group_version:p.group_version,client_request_id:newRequestId()})}>{t('splitProposeSettlement')}</PrimaryButton>:null}</Card>;})}
  {plan.data?.proposals.filter(p=>[p.user_a,p.user_b].includes(me?.id??'')).map(p=>{const mine=p.user_a===me?.id?p.a_accepted:p.b_accepted;const active=['proposed','agreed','reported'].includes(p.status);return <Card key={p.id} className="mb-2 gap-3 p-4">{text(p.pair)}<p className="text-caption text-ink-2">{t(status[p.status])}</p>
   {p.status==='proposed'&&!mine?<PrimaryButton disabled={action.isPending} onClick={()=>run({id:p.id,action:'agree'})}>{t('splitAcceptSettlement')}</PrimaryButton>:null}
   {p.status==='agreed'&&p.pair.debtor_id===me?.id?<PrimaryButton disabled={action.isPending} onClick={()=>run({id:p.id,action:'report'})}>{t('splitSent')}</PrimaryButton>:null}
   {['agreed','reported'].includes(p.status)&&p.pair.creditor_id===me?.id?<PrimaryButton disabled={action.isPending} onClick={()=>run({id:p.id,action:'confirm'})}>{t('splitReceived')}</PrimaryButton>:null}
   {active?<SecondaryButton disabled={action.isPending} onClick={()=>run({id:p.id,action:'reject'})}>{t('cancel')}</SecondaryButton>:null}
  </Card>;})}
  {plan.data&&!plan.data.pairs.some(related)&&!plan.data.proposals.some(p=>[p.user_a,p.user_b].includes(me?.id??''))?<p className="py-3 text-ink-2">{t('splitNothingToSettle')}</p>:null}
 </Screen>;
}
