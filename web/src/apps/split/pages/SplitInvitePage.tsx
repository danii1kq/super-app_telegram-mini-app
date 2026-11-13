import { useQuery, useMutation } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { api, ApiError } from '@/shared/api/client';
import { AppHeader, Banner, Card, PrimaryButton, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { useFeedback } from '@/shared/ui/toast';

interface Invite {group_title:string;group_id:string;receipt_id:string;joined:boolean}
export function SplitInvitePage(){
 const {token=''}=useParams(),navigate=useNavigate(),t=useT(),feedback=useFeedback();
 const info=useQuery({queryKey:['split','invite',token],queryFn:()=>api<Invite>(`/split/invites/${encodeURIComponent(token)}`),retry:false});
 const join=useMutation({mutationFn:()=>api<Invite>(`/split/invites/${encodeURIComponent(token)}/join`,{method:'POST',body:{}}),onSuccess:r=>navigate(`/apps/split/${r.group_id}/receipts/${r.receipt_id}`,{replace:true}),onError:e=>feedback(e instanceof ApiError?e.message:t('error'),'error')});
 return <Screen header={<AppHeader title={t('splitInvite')} onBack={()=>navigate('/apps/split')}/>}>
  {info.isError?<Banner icon="error" tone="danger" title={t('error')}>{t('splitInviteExpired')}</Banner>:null}
  {info.data?<Card className="mt-2 gap-4 p-4"><p className="text-headline-sm text-ink">{info.data.group_title}</p><p className="text-body-secondary text-ink-2">{t('splitJoinPrivacy')}</p><PrimaryButton full progress={join.isPending} onClick={()=>info.data.joined?navigate(`/apps/split/${info.data.group_id}/receipts/${info.data.receipt_id}`):join.mutate()}>{t(info.data.joined?'splitOpenReceipt':'splitJoinChoose')}</PrimaryButton></Card>:null}
 </Screen>;
}
