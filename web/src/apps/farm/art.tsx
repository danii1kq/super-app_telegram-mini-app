import type { CSSProperties } from 'react';

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
};

export function Glyph({ name, size = 20, className = '' }: { name: string; size?: number; className?: string }) {
  return <svg className={className} width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d={paths[name] ?? paths.sprout} /></svg>;
}

export function CropArt({ seed = 'carrot', mutation = 'none', growing = false, className = '' }: { seed?: string; mutation?: string; growing?: boolean; className?: string }) {
  const style: CSSProperties = { filter: mutation === 'gold' ? 'sepia(.8) saturate(2)' : mutation === 'crystal' ? 'hue-rotate(155deg) saturate(.8)' : undefined };
  return <svg className={`farm-crop-art ${className}`} style={style} viewBox="0 0 120 120" fill="none" aria-hidden="true">
    <ellipse cx="60" cy="104" rx="32" ry="7" fill="#213b21" opacity=".12" />
    {growing ? <><path d="M61 101V65" stroke="#457b32" strokeWidth="5" strokeLinecap="round"/><path d="M61 82C37 85 30 70 31 59c20-2 31 8 30 23Z" fill="#7eb54d"/><path d="M61 71c-1-20 12-30 30-27-1 17-10 28-30 27Z" fill="#4a8e44"/><path d="M65 65 81 53M57 77 40 66" stroke="#c7e68e" strokeWidth="2" strokeLinecap="round"/></> : <>
    {seed === 'carrot' && <><path d="M56 44C36 20 41 9 47 10c7 1 12 20 14 32-1-28 7-40 14-36s0 24-9 38c15-22 27-23 30-16s-14 15-30 22" fill="#538d40"/><path d="M40 48c4-13 35-13 39 0s-12 47-23 56c-5 3-24-41-16-56Z" fill="#ee9247"/><path d="M47 48c3-6 13-7 15-5" stroke="#ffbd70" strokeWidth="7" strokeLinecap="round"/><path d="m42 62 13 3m19 7-11 3m-15 9 10 2" stroke="#cf6e35" strokeWidth="3" strokeLinecap="round"/></>}
    {seed === 'tomato' && <><path d="M61 37c-5-13 3-20 12-22" stroke="#497b38" strokeWidth="5" strokeLinecap="round"/><path d="M61 43C24 23 15 65 28 88c11 23 54 21 65-2 11-25-2-58-32-43Z" fill="#e76b52"/><path d="M38 54c-10 7-11 20-7 27" stroke="#ffae89" strokeWidth="8" strokeLinecap="round"/><path d="m61 41-18-7 6 11-13 9 21-5 8 13 4-14 16 3-10-10 6-8-20 8Z" fill="#54894a"/></>}
    {seed === 'strawberry' && <><path d="M60 40C25 25 17 53 30 76s25 32 30 32 24-15 34-38-2-44-34-30Z" fill="#dc5d61"/><path d="M40 51c-8 3-10 10-7 16" stroke="#f69b97" strokeWidth="7" strokeLinecap="round"/><path d="m60 41-21-14 6 17-14 4 24 5 8 12 7-17 19-7-16-3 5-13-18 16Z" fill="#538748"/>{[[43,65],[65,65],[80,59],[54,81],[73,81],[62,96],[37,77]].map(([x,y],i)=><ellipse key={i} cx={x} cy={y} rx="2" ry="3" fill="#f7d799"/>)}</>}
    {seed === 'pumpkin' && <><path d="M60 40c-6-11-3-17 5-22" stroke="#6c813c" strokeWidth="8" strokeLinecap="round"/><path d="M66 32c16-14 25-7 25-7-6 14-19 14-25 7Z" fill="#70934b"/><ellipse cx="60" cy="75" rx="44" ry="31" fill="#db8739"/><ellipse cx="60" cy="74" rx="28" ry="33" fill="#eea147"/><ellipse cx="60" cy="74" rx="12" ry="33" fill="#f4b658"/><path d="M29 58c-7 6-9 15-6 22" stroke="#f7b862" strokeWidth="4" strokeLinecap="round"/></>}
    {seed === 'blueberry' && <><path d="M61 67V27m0 13L43 29m18 7 16-17" stroke="#617c44" strokeWidth="4"/><path d="M61 34c-17-2-24-16-20-23 16 0 23 10 20 23Z" fill="#7da15b"/><path d="M65 35c0-17 17-25 26-20-1 16-14 23-26 20Z" fill="#508652"/>{[[43,66],[77,66],[61,91]].map(([x,y],i)=><g key={i}><circle cx={x} cy={y} r="22" fill={i===1?'#6879b5':'#7884be'}/><path d={`m${x! - 6} ${y! - 13} 5 4 5-5-1 6 5 3-7 1-3-3-5 2 2-5Z`} fill="#4d5d95"/><circle cx={x! - 9} cy={y! - 5} r="4" fill="#b2bce0"/></g>)}</>}
    {seed === 'dragonfruit' && <><path d="M60 106C17 97 18 60 41 35L59 15l20 24c25 28 20 60-19 67Z" fill="#d8689c"/><path d="M60 102C30 91 34 62 59 37c28 29 28 53 1 65Z" fill="#f4ddcd"/><path d="m41 37-14 3 6 16-13 5 10 15-7 12 15 1m39-50 15 2-6 15 14 5-9 15 6 12-15 2M59 16l-4 17 12-5" stroke="#79a25d" strokeWidth="5" strokeLinejoin="round"/>{[[56,59],[66,70],[51,77],[65,89],[49,92]].map(([x,y],i)=><ellipse key={i} cx={x} cy={y} rx="1.5" ry="2.5" fill="#65574c"/>)}</>}
    </>}
    {mutation !== 'none' && <g fill={mutation === 'crystal' ? '#c3a4f0' : '#ecc55c'}><path d="m100 24 3 7 7 3-7 3-3 7-3-7-7-3 7-3 3-7Z"/><path d="m19 40 2 4 4 2-4 2-2 4-2-4-4-2 4-2 2-4Z"/></g>}
  </svg>;
}

export function SeedPacket({ exotic = false }: { exotic?: boolean }) {
  return <div className={`farm-packet ${exotic ? 'exotic' : ''}`}><div className="farm-packet-stitch"/><span className="farm-packet-label">SRU SEEDS</span><CropArt seed={exotic ? 'dragonfruit' : 'carrot'}/><span className="farm-packet-bottom">{exotic ? 'EXOTIC BLEND' : 'GARDEN MIX'}<small>3 SEEDS · GROW SOMETHING GOOD</small></span></div>;
}
