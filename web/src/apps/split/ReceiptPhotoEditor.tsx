import { useEffect, useRef, useState } from 'react';
import { Card, SecondaryButton } from '@/shared/ui';
import { useT } from '@/shared/i18n';
import { cropPixels, rotatedSize, type Crop } from './photo';

const full: Crop = { x: 0, y: 0, width: 100, height: 100 };

export function ReceiptPhotoEditor({ file, onChange, onEditing }: { file: File; onChange: (file: File) => void; onEditing: (editing: boolean) => void }) {
  const t = useT();
  const canvas = useRef<HTMLCanvasElement>(null);
  const drag = useRef<{ x: number; y: number } | null>(null);
  const [turns, setTurns] = useState(0);
  const [crop, setCrop] = useState<Crop>(full);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState(false);
  const [encoding, setEncoding] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    let active = true;
    setReady(false); setError(false);
    void (async () => {
      try {
        const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
        try {
          if (!active || !canvas.current) return;
          if (bitmap.width * bitmap.height > 40_000_000) throw new Error('pixels');
          const size = rotatedSize(bitmap.width, bitmap.height, turns);
          const target = canvas.current;
          target.width = size.width; target.height = size.height;
          const ctx = target.getContext('2d');
          if (!ctx) throw new Error('canvas');
          ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, target.width, target.height);
          ctx.translate(target.width / 2, target.height / 2); ctx.rotate(turns * Math.PI / 2);
          const w = turns % 2 ? size.height : size.width, h = turns % 2 ? size.width : size.height;
          ctx.drawImage(bitmap, -w / 2, -h / 2, w, h);
          setReady(true);
        } finally { bitmap.close(); }
      } catch { if (active) setError(true); }
    })();
    return () => { active = false; generation.current += 1; };
  }, [file, turns]);

  const change = (next: Crop) => { setCrop(next); onEditing(true); };
  const point = (event: React.PointerEvent<HTMLDivElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    return { x: Math.max(0, Math.min(100, (event.clientX - rect.left) / rect.width * 100)), y: Math.max(0, Math.min(100, (event.clientY - rect.top) / rect.height * 100)) };
  };
  const apply = () => {
    const source = canvas.current;
    if (!ready || !source || encoding) return;
    setEncoding(true);
    const current = generation.current;
    const area = cropPixels(source.width, source.height, crop);
    const output = document.createElement('canvas'); output.width = area.width; output.height = area.height;
    const ctx = output.getContext('2d');
    if (!ctx) { setEncoding(false); setError(true); return; }
    ctx.drawImage(source, area.x, area.y, area.width, area.height, 0, 0, area.width, area.height);
    output.toBlob((blob) => {
      output.width = 0; output.height = 0;
      if (current !== generation.current) return;
      setEncoding(false);
      if (!blob || blob.size > 20 * 1024 * 1024) { setError(true); return; }
      onChange(new File([blob], 'receipt.jpg', { type: 'image/jpeg' })); onEditing(false);
    }, 'image/jpeg', 0.92);
  };

  return <Card className="mt-2 flex flex-col gap-3 p-3">
    <p className="text-caption text-ink-2">{t('splitCropHint')}</p>
    <div className="relative touch-none overflow-hidden rounded-xl" onPointerDown={(e) => { if (!ready || encoding) return; drag.current = point(e); e.currentTarget.setPointerCapture(e.pointerId); }} onPointerMove={(e) => { if (!drag.current) return; const p = point(e), start = drag.current; change({ x: Math.min(start.x, p.x), y: Math.min(start.y, p.y), width: Math.max(1, Math.abs(start.x - p.x)), height: Math.max(1, Math.abs(start.y - p.y)) }); }} onPointerUp={() => { drag.current = null; }} onPointerCancel={() => { drag.current = null; }}>
      <canvas ref={canvas} className="block h-auto w-full" role="img" aria-label={t('receiptPhotoPreview')} />
      {ready ? <div className="pointer-events-none absolute border-2 border-white outline outline-1 outline-black/50" style={{ left: `${crop.x}%`, top: `${crop.y}%`, width: `${crop.width}%`, height: `${crop.height}%`, boxShadow: '0 0 0 100vmax #0006' }} /> : null}
    </div>
    {error ? <p role="alert" className="text-caption text-danger">{t('splitCropError')}</p> : null}
    <fieldset disabled={!ready || encoding} className="grid grid-cols-2 gap-2">
      {(['x', 'y', 'width', 'height'] as const).map((key) => <label key={key} className="text-caption text-ink-2">{t(({ x: 'splitCropLeft', y: 'splitCropTop', width: 'splitCropWidth', height: 'splitCropHeight' } as const)[key])}
        <input aria-label={t(({ x: 'splitCropLeft', y: 'splitCropTop', width: 'splitCropWidth', height: 'splitCropHeight' } as const)[key])} type="range" min={key === 'width' || key === 'height' ? 1 : 0} max={key === 'width' ? 100 - crop.x : key === 'height' ? 100 - crop.y : 99} value={crop[key]} onChange={(e) => { const next = { ...crop, [key]: Number(e.target.value) }; next.width = Math.min(next.width, 100 - next.x); next.height = Math.min(next.height, 100 - next.y); change(next); }} className="block w-full accent-accent" />
      </label>)}
    </fieldset>
    <div className="flex flex-wrap gap-2">
      <SecondaryButton disabled={!ready || encoding} onClick={() => { setTurns((v) => (v + 3) % 4); change(full); }}>{t('splitRotateLeft')}</SecondaryButton>
      <SecondaryButton disabled={!ready || encoding} onClick={() => { setTurns((v) => (v + 1) % 4); change(full); }}>{t('splitRotateRight')}</SecondaryButton>
      <SecondaryButton disabled={encoding} onClick={() => { setTurns(0); setCrop(full); onChange(file); onEditing(false); }}>{t('splitPhotoOriginal')}</SecondaryButton>
      <SecondaryButton disabled={!ready || encoding} onClick={apply}>{t('splitCropApply')}</SecondaryButton>
    </div>
  </Card>;
}
