import { useEffect, useRef, useState } from 'react';
import type { Attachment } from './api';
import { apiBlob } from '@/shared/api/client';
import { Icon } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { useFeedback } from '@/shared/ui/toast';

export function MessageAttachment({ conversationId, messageId, attachment }: { conversationId: string; messageId: string; attachment: Attachment }) {
  const t = useT();
  const feedback = useFeedback();
  const element = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);
  const [url, setUrl] = useState('');
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  const [downloading, setDownloading] = useState(false);
  const path = `/conversations/${conversationId}/messages/${messageId}/attachment`;

  useEffect(() => {
    if (!element.current || attachment.kind !== 'photo') return;
    const observer = new IntersectionObserver(([entry]) => {
      if (entry?.isIntersecting) { setVisible(true); observer.disconnect(); }
    }, { rootMargin: '200px' });
    observer.observe(element.current);
    return () => observer.disconnect();
  }, [attachment.kind]);

  useEffect(() => {
    if (!visible || attachment.kind !== 'photo') return;
    const controller = new AbortController();
    let objectUrl = '';
    setFailed(false);
    void apiBlob(path, controller.signal).then((blob) => {
      if (controller.signal.aborted) return;
      objectUrl = URL.createObjectURL(blob);
      setUrl(objectUrl);
    }).catch(() => { if (!controller.signal.aborted) setFailed(true); });
    return () => { controller.abort(); if (objectUrl) URL.revokeObjectURL(objectUrl); };
  }, [visible, attachment.kind, path, retry]);

  const download = async () => {
    if (downloading) return;
    setDownloading(true);
    let downloadUrl = '';
    try {
      downloadUrl = url || URL.createObjectURL(await apiBlob(path));
      const link = document.createElement('a');
      link.href = downloadUrl; link.download = attachment.name;
      document.body.append(link); link.click(); link.remove();
    } catch { feedback(t('chatAttachmentFailed'), 'error'); }
    finally {
      setDownloading(false);
      if (downloadUrl && !url) setTimeout(() => URL.revokeObjectURL(downloadUrl), 1000);
    }
  };

  return <div ref={element} className="py-1">
    {attachment.kind === 'photo' ? <div className="relative w-full min-w-40 overflow-hidden rounded-xl bg-neutral-soft/30" style={{ aspectRatio: `${attachment.width ?? 4}/${attachment.height ?? 3}`, maxHeight: 360 }}>
      {url ? <img src={url} alt={t('chatPhoto')} width={attachment.width} height={attachment.height} className="h-full w-full object-contain" /> : <div className="flex h-full items-center justify-center text-caption" role="status">{failed ? <button type="button" onClick={() => setRetry((n) => n + 1)} className="min-h-11 px-3">{t('retry')}</button> : t('loading')}</div>}
    </div> : null}
    <button type="button" onClick={() => void download()} disabled={downloading} className="flex min-h-11 max-w-full items-center gap-2 text-left text-caption disabled:opacity-50">
      <Icon name={downloading ? 'hourglass_top' : 'download'} size={20} />
      <span className="min-w-0 break-words [overflow-wrap:anywhere]">{attachment.kind === 'photo' ? t('chatDownload') : attachment.name}<span className="block opacity-70">{Math.max(1, Math.ceil(attachment.size / 1024))} KB</span></span>
    </button>
  </div>;
}
