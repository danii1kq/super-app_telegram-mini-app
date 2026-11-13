import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useManualReceipt, useUploadReceipt } from '../receipts';
import { ReceiptPhotoEditor } from '../ReceiptPhotoEditor';
import { AppHeader, Card, PrimaryButton, SecondaryButton, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { useFeedback } from '@/shared/ui/toast';
import { ApiError } from '@/shared/api/client';

export function SplitReceiptPage() {
 const {groupId=''}=useParams();const navigate=useNavigate();const t=useT();const feedback=useFeedback();
 const upload=useUploadReceipt(groupId), manual=useManualReceipt(groupId);
 const [source,setSource]=useState<File|null>(null),[selected,setSelected]=useState<File|null>(null),[preview,setPreview]=useState('');
 const [editing,setEditing]=useState(false),[consent,setConsent]=useState(false);
 const input=useRef<HTMLInputElement>(null);const busy=upload.isPending||manual.isPending;
 useEffect(()=>{if(!selected){setPreview('');return;}const url=URL.createObjectURL(selected);setPreview(url);return()=>URL.revokeObjectURL(url);},[selected]);
 const failed=(error:unknown)=>feedback(error instanceof ApiError?error.message:t('receiptUploadFailed'),'error');
 return <Screen header={<AppHeader title={t('receiptAdd')} onBack={()=>navigate(`/apps/split/${groupId}`)}/> }>
  <Card className="mt-2 gap-3 p-4">
   <p className="text-caption text-ink-2">{t('receiptAddHint')}</p>
   <input ref={input} type="file" accept="image/jpeg,image/png,image/webp" className="hidden" onChange={(event)=>{const file=event.target.files?.[0];event.target.value='';if(!file)return;if(!file.size||file.size>20*1024*1024||!['image/jpeg','image/png','image/webp'].includes(file.type)){feedback(t('receiptImageLimit'),'error');return;}setSource(file);setSelected(file);setEditing(false);setConsent(false);}}/>
   <PrimaryButton full disabled={busy} onClick={()=>input.current?.click()}>{source?t('receiptChooseAnother'):t('receiptTakePhoto')}</PrimaryButton>
   <SecondaryButton full disabled={busy} onClick={()=>manual.mutate(undefined,{onSuccess:r=>navigate(`/apps/split/${groupId}/receipts/${r.id}`),onError:failed})}>{t('receiptManual')}</SecondaryButton>
  </Card>
  {source?<fieldset disabled={busy} className={busy?'pointer-events-none opacity-60':''}><ReceiptPhotoEditor key={`${source.name}:${source.lastModified}`} file={source} onChange={file=>{setSelected(file);setConsent(false);}} onEditing={setEditing}/></fieldset>:null}
  {preview?<Card className="mt-2 gap-3 p-3"><img src={preview} alt={t('receiptPhotoPreview')} className="max-h-72 w-full rounded-xl object-contain"/>
   <p className="text-caption text-ink-2">{t('receiptLocalPreview')}</p>
   <label className="flex min-h-11 items-start gap-2 text-caption text-ink-2"><input type="checkbox" disabled={busy} checked={consent} onChange={e=>setConsent(e.target.checked)} className="mt-1 size-5 shrink-0 accent-accent"/>{t('receiptDeepSeekConsent')}</label>
   <PrimaryButton full progress={upload.isPending} disabled={!consent||editing||busy} onClick={()=>{if(selected)upload.mutate(selected,{onSuccess:r=>navigate(`/apps/split/${groupId}/receipts/${r.id}`),onError:failed});}}>{t('receiptRecognize')}</PrimaryButton>
  </Card>:null}
 </Screen>;
}
