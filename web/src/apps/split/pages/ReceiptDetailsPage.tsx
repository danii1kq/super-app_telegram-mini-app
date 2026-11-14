import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { api, ApiError } from '@/shared/api/client';
import { useSession } from '@/app/session';
import { AppHeader, Banner, Card, PrimaryButton, SecondaryButton, Screen } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useT, type DictKey } from '@/shared/i18n';
import { formatMinor } from '../api';
import { newRequestId } from '../mutations';
import { parseReceiptMinor, receiptPreview } from '../receiptPreview';
import { useGroupMembers, useGroupPeople, useReceipt, useReceiptAction, useUpdateReceipt, type Receipt, type ReceiptItem } from '../receipts';

const statusKeys: Record<Receipt['status'],DictKey>={draft:'splitDraft',collecting:'splitCollecting',review:'splitReview',saved:'splitSaved',cancelled:'splitCancelled'};
const field='min-h-11 min-w-0 rounded-lg bg-neutral-soft px-3 py-2 text-body-lg text-ink';
type Editable=ReceiptItem & {price:string;localKey:string};

export function ReceiptDetailsPage(){
 const {groupId='',receiptId=''}=useParams();const navigate=useNavigate();const t=useT();const feedback=useFeedback();const {me}=useSession();
 const query=useReceipt(receiptId),actions=useReceiptAction(receiptId,groupId),update=useUpdateReceipt(receiptId),people=useGroupPeople(groupId),members=useGroupMembers(groupId);
 const [edit,setEdit]=useState<{version:number;items:Editable[];total:string;discount:string;service:string}|null>(null);
 const [payer,setPayer]=useState(''),[question,setQuestion]=useState(''),[link,setLink]=useState(''),[linkBusy,setLinkBusy]=useState(false),[reviewed,setReviewed]=useState(false);
 const receipt=query.data;const busy=actions.isPending||update.isPending||linkBusy;
 const owner=receipt?.author_id===me?.id;
 const name=(id:string)=>id===me?.id?t('you'):people.data?.find(p=>p.id===id)?.public_name??`${t('member')} ${id}`;
 const payerId=payer||receipt?.payer_id||receipt?.author_id||'';
 const preview=useMemo(()=>receipt?receiptPreview(receipt.items.map(i=>({key:i.id,amount:Number(i.amount_minor),units:i.units})),Number(receipt.discount_minor),Number(receipt.service_minor),Number(receipt.total_minor),payerId):null,[receipt,payerId]);
 const failed=(error:unknown)=>{feedback(error instanceof ApiError?error.message:t('receiptSaveFailed'),'error');if(error instanceof ApiError&&error.status===409)setEdit(null);};
 const act=(action:string,body:Record<string,unknown>={})=>{if(!receipt)return;actions.mutate({action,version:receipt.version,...body},{onError:failed});};
 const beginEdit=()=>{if(!receipt)return;setReviewed(false);setEdit({version:receipt.version,items:receipt.items.map(i=>({...i,units:{...i.units},price:formatMinor(i.amount_minor),localKey:i.id})),total:formatMinor(receipt.total_minor),discount:formatMinor(receipt.discount_minor),service:formatMinor(receipt.service_minor)});};
 const patch=(index:number,values:Partial<Editable>)=>{setReviewed(false);setEdit(e=>e?{...e,items:e.items.map((item,i)=>i===index?{...item,...values}:item)}:e);};
 const saveEdit=()=>{
  if(!edit||!reviewed)return;
  const amounts=[edit.total,edit.discount,edit.service].map(parseReceiptMinor);
  if(amounts.some(v=>v===null)||edit.items.some(i=>parseReceiptMinor(i.price)===null)){feedback(t('checkAmount'),'error');return;}
  update.mutate({version:edit.version,totalMinor:String(amounts[0]),discountMinor:String(amounts[1]),serviceMinor:String(amounts[2]),items:edit.items.map(i=>({id:i.id||undefined,name:i.name,quantity:i.quantity,amountMinor:String(parseReceiptMinor(i.price)),unclear:false,units:i.units,allocationMode:i.allocation_mode,parentPosition:i.parent_position}))},{onSuccess:()=>setEdit(null),onError:failed});
 };
 const share=async()=>{setLinkBusy(true);try{const result=await api<{url:string}>(`/split/receipts/${receiptId}/invite`,{method:'POST',body:{}});if(!result.url){feedback(t('splitLinkUnavailable'),'error');return;}setLink(result.url);}catch(error){failed(error);}finally{setLinkBusy(false);}};
 if(!receipt)return <Screen header={<AppHeader title={t('receiptTitle')} onBack={()=>navigate(`/apps/split/${groupId}`)}/>}>{query.isError?<Banner icon="error" tone="danger" title={t('error')}>{t('retryConnection')}</Banner>:<p className="py-4 text-ink-2">{t('loading')}</p>}<SecondaryButton onClick={()=>void query.refetch()}>{t('refresh')}</SecondaryButton></Screen>;
 const canEdit=owner&&(receipt.status==='draft'||receipt.status==='collecting');
 const myApproval=receipt.approvals.find(a=>a.user_id===me?.id);
 return <Screen header={<AppHeader title={t('receiptTitle')} onBack={()=>navigate(`/apps/split/${groupId}`)}/>} bottomSpace>
  <Card className="mt-2 gap-2 p-4"><p className="text-body-strong text-ink">{t(statusKeys[receipt.status])}</p><p className="text-headline-md text-ink">{formatMinor(receipt.total_minor)} ₸</p>{receipt.image_state!=='none'?<p className="text-caption text-ink-2">{t('splitPhotoNotStored')}</p>:null}{receipt.ocr_error?<p className="text-caption text-warning">{t('receiptOcrFailed')}</p>:null}</Card>
  {canEdit&&!edit?<SecondaryButton disabled={busy} onClick={beginEdit}>{t('splitEditReceipt')}</SecondaryButton>:null}
  {edit?<fieldset disabled={busy} className="mt-2 flex flex-col gap-3">
   {edit.items.map((item,index)=><Card key={item.localKey} className="gap-2 p-3">
    <input aria-label={t('receiptItemName')} className={field} maxLength={200} value={item.name} onChange={e=>patch(index,{name:e.target.value})}/>
    <div className="flex gap-2"><input aria-label={t('receiptQuantity')} className={`${field} w-24`} type="number" min={1} max={999} value={item.quantity} onChange={e=>patch(index,{quantity:Number(e.target.value)})}/><input aria-label={t('expenseAmount')} className={`${field} w-full`} inputMode="decimal" value={item.price} onChange={e=>patch(index,{price:e.target.value})}/></div>
    <label className="text-caption text-ink-2">{t('splitComboParent')}<select className={`${field} w-full`} value={item.parent_position??''} onChange={e=>patch(index,{parent_position:e.target.value===''?null:Number(e.target.value)})}><option value="">{t('splitSeparateItem')}</option>{edit.items.map((root,i)=>i!==index&&root.parent_position==null?<option key={root.localKey} value={i}>{root.name||String(i+1)}</option>:null)}</select></label>
    {item.suggested_parent_position!=null&&item.parent_position==null?<SecondaryButton onClick={()=>patch(index,{parent_position:item.suggested_parent_position!})}>{t('splitSuggestedCombo')}</SecondaryButton>:null}
    {item.parent_position==null?<><label className="text-caption text-ink-2">{t('splitSelectionMode')}<select className={`${field} w-full`} value={item.allocation_mode} onChange={e=>patch(index,{allocation_mode:e.target.value as 'units'|'shared',units:{}})}><option value="units">{t('splitIndividual')}</option><option value="shared">{t('splitShared')}</option></select></label>
     <div className="flex flex-wrap gap-2">{members.data?.map(id=><label key={id} className="flex min-h-11 items-center gap-2 text-caption text-ink-2">{name(id)}<input aria-label={`${t('receiptQuantity')}: ${name(id)}`} type="number" min={0} max={item.allocation_mode==='shared'?1:item.quantity} className={`${field} w-20`} value={item.units[id]??0} onChange={e=>{const units={...item.units};const n=Number(e.target.value);if(n===0)delete units[id];else units[id]=n;patch(index,{units});}}/></label>)}</div>
    </>:null}
    <SecondaryButton onClick={()=>{setReviewed(false);setEdit(e=>e?{...e,items:e.items.filter((_,i)=>i!==index).map(i=>({...i,parent_position:i.parent_position==null||i.parent_position===index?null:i.parent_position>index?i.parent_position-1:i.parent_position}))}:e);}}>{t('delete')}</SecondaryButton>
   </Card>)}
   <SecondaryButton disabled={edit.items.length>=200} onClick={()=>{setReviewed(false);setEdit(e=>e?{...e,items:[...e.items,{id:'',position:e.items.length,name:'',quantity:1,amount_minor:'0',price:'0,00',unclear:false,units:{},allocation_mode:'units',parent_position:null,localKey:newRequestId()}]}:e);}}>{t('splitAddItem')}</SecondaryButton>
   {(['total','discount','service'] as const).map(key=><label key={key} className="text-caption text-ink-2">{t(({total:'receiptTotal',discount:'receiptDiscount',service:'receiptService'} as const)[key])}<input className={`${field} w-full`} inputMode="decimal" value={edit[key]} onChange={e=>{setReviewed(false);setEdit(d=>d?{...d,[key]:e.target.value}:d);}}/></label>)}
   <label className="flex min-h-11 items-center gap-2 text-caption text-ink-2"><input type="checkbox" className="size-5 accent-accent" checked={reviewed} onChange={e=>setReviewed(e.target.checked)}/>{t('splitCheckedPrices')}</label>
   <PrimaryButton full disabled={!reviewed} progress={update.isPending} onClick={saveEdit}>{t('save')}</PrimaryButton><SecondaryButton onClick={()=>setEdit(null)}>{t('cancel')}</SecondaryButton>
  </fieldset>:<div className="mt-2 flex flex-col gap-2">{receipt.items.filter(i=>i.parent_position==null).map(item=>{
   const children=receipt.items.filter(i=>i.parent_position===item.position),mine=item.units[me?.id??'']??0,totalUnits=Object.values(item.units).reduce((a,b)=>a+b,0);
   return <Card key={item.id} className="gap-2 p-3"><div className="flex items-start justify-between gap-2"><span className="text-body-strong text-ink">{item.name}{item.unclear?' ?':''}</span><span className="shrink-0 text-body-strong text-ink">{formatMinor(String(BigInt(item.amount_minor)+children.reduce((sum,i)=>sum+BigInt(i.amount_minor),0n)))} ₸</span></div>
    {children.length?<details><summary className="min-h-11 py-2 text-caption text-ink-2">{t('splitComboContents')} ({children.length})</summary>{children.map(child=><p key={child.id} className="py-1 text-caption text-ink-2">{child.name} · {formatMinor(child.amount_minor)} ₸</p>)}</details>:null}
    <p className="text-caption text-ink-2">{item.allocation_mode==='shared'?t('splitShared'):`${totalUnits}/${item.quantity}`} · {Object.keys(item.units).map(name).join(', ')||t('splitNobodyYet')}</p>
    {receipt.status==='collecting'?<div className="flex items-center gap-3">{item.allocation_mode==='shared'?<SecondaryButton disabled={busy} onClick={()=>act('claim',{item_id:item.id,count:mine?0:1})}>{mine?t('splitLeaveItem'):t('splitTakeItem')}</SecondaryButton>:<><SecondaryButton disabled={busy||mine===0} onClick={()=>act('claim',{item_id:item.id,count:mine-1})}>-</SecondaryButton><span className="text-body-strong text-ink">{mine}</span><SecondaryButton disabled={busy||totalUnits>=item.quantity} onClick={()=>act('claim',{item_id:item.id,count:mine+1})}>+</SecondaryButton></>}</div>:null}
   </Card>;
  })}</div>}
  {!edit&&owner&&receipt.status==='draft'?<PrimaryButton full disabled={busy} onClick={()=>act('collect')}>{t('splitOpenSelection')}</PrimaryButton>:null}
  {!edit&&receipt.status==='collecting'?<Card className="mt-2 gap-3 p-3">
   {owner?<><SecondaryButton disabled={busy} onClick={()=>void share()}>{t('splitInvite')}</SecondaryButton>{link?<><input readOnly aria-label={t('splitInvite')} className={field} value={link}/><SecondaryButton onClick={()=>void navigator.clipboard.writeText(link).then(()=>feedback(t('splitLinkCopied'))).catch(()=>feedback(t('splitCopyManually'),'error'))}>{t('splitCopyLink')}</SecondaryButton></>:null}<SecondaryButton disabled={busy} onClick={()=>{setLinkBusy(true);void api(`/split/receipts/${receiptId}/revoke-invites`,{method:'POST',body:{}}).then(()=>{setLink('');feedback(t('splitLinksRevoked'));}).catch(failed).finally(()=>setLinkBusy(false));}}>{t('splitRevokeLinks')}</SecondaryButton></>:<p className="text-caption text-ink-2">{t('splitChooseOwn')}</p>}
  </Card>:null}
  {preview?<Card className="mt-2 gap-2 p-3"><p className="text-body-strong text-ink">{t('splitYourShare')}: {formatMinor(String(preview.perPerson[me?.id??'']??0))} ₸</p><p className="text-caption text-ink-2">{t('receiptDiscount')}: {formatMinor(receipt.discount_minor)} ₸ · {t('receiptService')}: {formatMinor(receipt.service_minor)} ₸</p>{preview.difference!==0?<p role="alert" className="text-caption text-danger">{t('splitDifference')}: {formatMinor(String(preview.difference))} ₸</p>:null}</Card>:null}
  {!edit&&canEdit?<Card className="mt-2 gap-3 p-3"><label className="text-caption text-ink-2">{t('receiptPayer')}<select className={`${field} w-full`} value={payerId} onChange={e=>setPayer(e.target.value)}>{members.data?.map(id=><option key={id} value={id}>{name(id)}</option>)}</select></label><PrimaryButton full disabled={busy||!receipt.items.length||!!preview?.unallocated.length||preview?.difference!==0||receipt.items.some(i=>i.unclear)} onClick={()=>act('submit',{payer_id:payerId,client_request_id:newRequestId(),description:receipt.description||t('receiptTitle'),category:receipt.category||'cafe'})}>{t('splitSubmitReview')}</PrimaryButton></Card>:null}
  {receipt.approvals.length?<Card className="mt-2 gap-2 p-3">{receipt.status==='review'?<p className="text-caption text-ink-2">{t('splitConsensusHint')}</p>:null}{receipt.approvals.map(a=><div key={a.user_id} className="text-caption text-ink"><p>{name(a.user_id)} · {t(a.decision==='accepted'?'splitAgreed':a.decision==='question'?'splitQuestion':'splitWaiting')}</p>{a.question?<p className="break-words text-ink-2">{a.question}</p>:null}</div>)}</Card>:null}
  {receipt.status==='review'&&myApproval?.decision==='pending'?<Card className="mt-2 gap-3 p-3"><PrimaryButton full disabled={busy} onClick={()=>act('respond',{decision:'accepted'})}>{t('splitAgree')}</PrimaryButton><textarea aria-label={t('splitQuestion')} className={field} maxLength={500} value={question} onChange={e=>setQuestion(e.target.value)}/><SecondaryButton disabled={busy||!question.trim()} onClick={()=>act('respond',{decision:'question',question})}>{t('splitQuestion')}</SecondaryButton></Card>:null}
  {owner&&(receipt.status==='review'||receipt.status==='collecting')?<SecondaryButton disabled={busy} onClick={()=>act('reopen')}>{t('splitReopen')}</SecondaryButton>:null}
  {owner&&receipt.status!=='saved'&&receipt.status!=='cancelled'?<SecondaryButton disabled={busy} onClick={()=>{if(window.confirm(t('splitCancelConfirm')))act('cancel');}}>{t('splitCancelReceipt')}</SecondaryButton>:null}
 </Screen>;
}
