import type { CSSProperties } from 'react';
import { GrowingPlant } from './GrowingPlant';

const paths: Record<string, string> = {
  sprout: 'M12 21v-9M12 16C4 16 3 11 3 7c6 0 9 3 9 9Zm0-4c0-6 3-9 9-9 0 6-3 9-9 9Z',
  garden: 'M3 21h18M5 17V9m14 8V9M3 9l9-6 9 6M8 21v-8h8v8',
  seed: 'M19 5C7 3 2 10 6 17s15 2 13-12ZM6 18 16 8',
  book: 'M12 6c-3-3-7-3-10-2v15c4-1 7 0 10 2 3-2 6-3 10-2V4c-3-1-7-1-10 2Zm0 0v15',
  trophy: 'M8 3h8v6a4 4 0 0 1-8 0V3Zm0 2H4v3c0 3 2 4 5 4m7-7h4v3c0 3-2 4-5 4m-3 1v6m-4 2h8',
  upgrade: 'm14 3 7 7-7 7m7-7H9a6 6 0 0 0-6 6v5',
  coin: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-7-4h-3a2 2 0 0 0 0 4h2a2 2 0 0 1 0 4h-3m2-10v2m0 8v2',
  arrow: 'm14 6-6 6 6 6',
  plus: 'M12 5v14M5 12h14',
  close: 'm6 6 12 12M6 18 18 6',
  water: 'M12 2S5 10 5 15a7 7 0 0 0 14 0c0-5-7-13-7-13Zm-3 13c0 2 1 3 3 3',
  soil: 'M5 20c-3-6 1-10 7-10s10 4 7 10H5Zm7-10V4m-4 0 4 3 4-3',
  lamp: 'M9 18h6m-5 3h4M8 14a6 6 0 1 1 8 0l-1 2H9l-1-2Z',
  clock: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-5v5l3 2',
  check: 'm5 12 4 4L19 6',
  box: 'm3 7 9-4 9 4v10l-9 4-9-4V7Zm0 0 9 4 9-4m-9 4v10M7 5l10 4',
  spark: 'm12 3 2.5 6.5L21 12l-6.5 2.5L12 21l-2.5-6.5L3 12l6.5-2.5L12 3Z',
  sun: 'M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM12 2v2m0 16v2M2 12h2m16 0h2M5 5l1 1m12 12 1 1M5 19l1-1M18 6l1-1',
  sound: 'M11 4 6 8H3v8h3l5 4V4Zm4 4c3 2 3 6 0 8m3-11c5 4 5 10 0 14',
  muted: 'M11 4 6 8H3v8h3l5 4V4Zm5 5 5 6m-5 0 5-6',
};

export function Glyph({ name, size = 20, className = '' }: { name: string; size?: number; className?: string }) {
  return <svg className={className} width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d={paths[name] ?? paths.sprout} /></svg>;
}

export function CropArt({ seed = 'carrot', mutation = 'none', growing = false, progress = 0, variant = 0, className = '' }: { seed?: string; mutation?: string; growing?: boolean; progress?: number; variant?: number; className?: string }) {
  const style: CSSProperties = { filter: mutation === 'gold' ? 'sepia(.8) saturate(2)' : mutation === 'crystal' ? 'hue-rotate(155deg) saturate(.8)' : undefined };
  return <svg className={`farm-crop-art ${className}`} style={style} viewBox="0 0 120 120" fill="none" aria-hidden="true">
    <ellipse cx="60" cy="104" rx="32" ry="7" fill="#213b21" opacity=".12" />
    {growing ? <GrowingPlant seed={seed} progress={progress} variant={variant}/> : <>
    {seed === 'carrot' && <><path d="M56 44C36 20 41 9 47 10c7 1 12 20 14 32-1-28 7-40 14-36s0 24-9 38c15-22 27-23 30-16s-14 15-30 22" fill="#538d40"/><path d="M40 48c4-13 35-13 39 0s-12 47-23 56c-5 3-24-41-16-56Z" fill="#ee9247"/><path d="M47 48c3-6 13-7 15-5" stroke="#ffbd70" strokeWidth="7" strokeLinecap="round"/><path d="m42 62 13 3m19 7-11 3m-15 9 10 2" stroke="#cf6e35" strokeWidth="3" strokeLinecap="round"/></>}
    {seed === 'tomato' && <><path d="M61 37c-5-13 3-20 12-22" stroke="#497b38" strokeWidth="5" strokeLinecap="round"/><path d="M61 43C24 23 15 65 28 88c11 23 54 21 65-2 11-25-2-58-32-43Z" fill="#e76b52"/><path d="M38 54c-10 7-11 20-7 27" stroke="#ffae89" strokeWidth="8" strokeLinecap="round"/><path d="m61 41-18-7 6 11-13 9 21-5 8 13 4-14 16 3-10-10 6-8-20 8Z" fill="#54894a"/></>}
    {seed === 'strawberry' && <><path d="M60 40C25 25 17 53 30 76s25 32 30 32 24-15 34-38-2-44-34-30Z" fill="#dc5d61"/><path d="M40 51c-8 3-10 10-7 16" stroke="#f69b97" strokeWidth="7" strokeLinecap="round"/><path d="m60 41-21-14 6 17-14 4 24 5 8 12 7-17 19-7-16-3 5-13-18 16Z" fill="#538748"/>{[[43,65],[65,65],[80,59],[54,81],[73,81],[62,96],[37,77]].map(([x,y],i)=><ellipse key={i} cx={x} cy={y} rx="2" ry="3" fill="#f7d799"/>)}</>}
    {seed === 'pumpkin' && <><path d="M60 40c-6-11-3-17 5-22" stroke="#6c813c" strokeWidth="8" strokeLinecap="round"/><path d="M66 32c16-14 25-7 25-7-6 14-19 14-25 7Z" fill="#70934b"/><ellipse cx="60" cy="75" rx="44" ry="31" fill="#db8739"/><ellipse cx="60" cy="74" rx="28" ry="33" fill="#eea147"/><ellipse cx="60" cy="74" rx="12" ry="33" fill="#f4b658"/><path d="M29 58c-7 6-9 15-6 22" stroke="#f7b862" strokeWidth="4" strokeLinecap="round"/></>}
    {seed === 'blueberry' && <><path d="M61 67V27m0 13L43 29m18 7 16-17" stroke="#617c44" strokeWidth="4"/><path d="M61 34c-17-2-24-16-20-23 16 0 23 10 20 23Z" fill="#7da15b"/><path d="M65 35c0-17 17-25 26-20-1 16-14 23-26 20Z" fill="#508652"/>{[[43,66],[77,66],[61,91]].map(([x,y],i)=><g key={i}><circle cx={x} cy={y} r="22" fill={i===1?'#6879b5':'#7884be'}/><path d={`m${x! - 6} ${y! - 13} 5 4 5-5-1 6 5 3-7 1-3-3-5 2 2-5Z`} fill="#4d5d95"/><circle cx={x! - 9} cy={y! - 5} r="4" fill="#b2bce0"/></g>)}</>}
    {seed === 'dragonfruit' && <><path d="M60 106C17 97 18 60 41 35L59 15l20 24c25 28 20 60-19 67Z" fill="#d8689c"/><path d="M60 102C30 91 34 62 59 37c28 29 28 53 1 65Z" fill="#f4ddcd"/><path d="m41 37-14 3 6 16-13 5 10 15-7 12 15 1m39-50 15 2-6 15 14 5-9 15 6 12-15 2M59 16l-4 17 12-5" stroke="#79a25d" strokeWidth="5" strokeLinejoin="round"/>{[[56,59],[66,70],[51,77],[65,89],[49,92]].map(([x,y],i)=><ellipse key={i} cx={x} cy={y} rx="1.5" ry="2.5" fill="#65574c"/>)}</>}
    {seed === 'radish' && <><path d="M57 45C30 21 40 8 46 12c7 5 10 17 14 29-2-28 12-35 15-26s-5 21-11 28c18-16 27-14 27-6s-18 12-27 11" fill="#69a45c"/><path d="M60 42c-38-8-43 36-10 50l10 16 4-18c34-18 26-52-4-48Z" fill="#d65e7c"/><path d="M43 79c6 2 20 6 34-1-3 6-7 10-13 12l-4 18-10-16c-5-2-9-7-7-13Z" fill="#fff1dc"/><path d="M43 53c-7 3-10 12-7 17" stroke="#ef9bac" strokeWidth="6" strokeLinecap="round"/></>}
    {seed === 'cucumber' && <><path d="M38 94c-16-15-1-48 24-70 17-15 37-4 30 16-5 18-28 68-44 64Z" fill="#648f4b"/><path d="M41 87c-4-14 12-42 30-57" stroke="#94b872" strokeWidth="9" strokeLinecap="round"/><path d="M79 23c-3-14 9-14 12-8s-6 12-7 4" stroke="#6a8a49" strokeWidth="3" strokeLinecap="round"/>{[[54,65],[65,47],[68,71],[46,89],[78,39],[57,86]].map(([x,y],i)=><circle key={i} cx={x} cy={y} r="2" fill="#c8d78e"/>)}</>}
    {seed === 'corn' && <><path d="M60 103C23 84 31 41 53 20c8-8 18-5 22 7 13 32 3 65-15 76Z" fill="#edc45e"/>{[0,1,2,3].map(col=>[0,1,2,3,4,5].map(row=><rect key={`${col}-${row}`} x={43+col*7} y={32+row*9} width="5" height="7" rx="2" fill={(row+col)%2?'#f7d97c':'#dba743'}/>))}<path d="M59 105C26 97 19 76 23 54c9 20 24 19 36 51Z" fill="#6c9b50"/><path d="M59 105c31-6 44-27 38-62-8 23-28 26-38 62Z" fill="#8cb364"/><path d="M61 98 84 64" stroke="#c3d391" strokeWidth="2" strokeLinecap="round"/></>}
    {seed === 'sunflower' && <><path d="M60 102V58" stroke="#6c9850" strokeWidth="5"/><path d="M59 87C35 87 31 76 32 66c17-1 27 9 27 21Zm4-9c0-18 13-24 27-20-2 13-12 21-27 20Z" fill="#82a85e"/>{[0,45,90,135,180,225,270,315].map(angle=><ellipse key={angle} cx="60" cy="24" rx="10" ry="19" transform={`rotate(${angle} 60 45)`} fill={angle%90?'#e9b64f':'#f2cc66'}/>)}<circle cx="60" cy="45" r="19" fill="#916e43"/><circle cx="60" cy="45" r="14" fill="#ab8550"/>{[[53,40],[64,37],[59,48],[69,49],[51,50]].map(([x,y],i)=><circle key={i} cx={x} cy={y} r="2" fill="#d0a96c"/>)}</>}
    {seed === 'watermelon' && <><path d="M59 29c-9-11 2-19 8-14s-6 11-4 3" stroke="#547e44" strokeWidth="4" strokeLinecap="round"/><ellipse cx="60" cy="68" rx="44" ry="37" fill="#7da957"/><path d="M37 37C19 62 28 89 40 101m14-69c-9 28-7 53 2 72m15-71c13 26 11 50 0 70m13-62c18 19 18 38 6 52" stroke="#4c843e" strokeWidth="8" strokeLinecap="round"/><path d="M30 63c0-7 3-12 7-16" stroke="#b6cc81" strokeWidth="5" strokeLinecap="round"/></>}
    {seed === 'moonflower' && <><path d="M60 104V58" stroke="#77969a" strokeWidth="4"/><path d="M58 88C36 89 29 78 33 65c15 0 25 10 25 23Zm5-9c-1-19 13-25 27-20-3 14-13 20-27 20Z" fill="#8caeb0"/>{[0,72,144,216,288].map(angle=><path key={angle} d="M60 56C36 43 45 15 60 8c14 11 22 35 0 48Z" transform={`rotate(${angle} 60 49)`} fill={angle%144?'#c6b4e4':'#e2d5f4'} stroke="#ad96ca" strokeWidth="1.5"/>)}<circle cx="60" cy="49" r="12" fill="#f3df9c"/><path d="m60 40 2 6 6 3-6 2-2 7-2-7-6-2 6-3 2-6Z" fill="#fff5d4"/><path d="m99 19 2 5 5 2-5 2-2 5-2-5-5-2 5-2 2-5Z" fill="#e5c779"/></>}
    </>}
    {mutation !== 'none' && <g fill={mutation === 'crystal' ? '#c3a4f0' : '#ecc55c'}><path d="m100 24 3 7 7 3-7 3-3 7-3-7-7-3 7-3 3-7Z"/><path d="m19 40 2 4 4 2-4 2-2 4-2-4-4-2 4-2 2-4Z"/></g>}
  </svg>;
}

export function SeedPacket({ exotic = false, variant }: { exotic?: boolean; variant?: string }) {
  const tone = variant ?? (exotic ? 'exotic' : 'garden');
  return <div className={`farm-packet ${tone}`}><div className="farm-packet-stitch"/><span className="farm-packet-label">SRU SEEDS</span><CropArt seed={tone === 'lunar' ? 'moonflower' : tone === 'exotic' ? 'dragonfruit' : 'carrot'}/><span className="farm-packet-bottom">{tone === 'lunar' ? 'LUNAR BLOOM' : tone === 'exotic' ? 'EXOTIC BLEND' : 'GARDEN MIX'}<small>3 SEEDS · GROW SOMETHING GOOD</small></span></div>;
}
