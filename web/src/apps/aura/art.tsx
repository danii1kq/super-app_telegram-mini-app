import { useId } from 'react';
import './art.css';

type ArtProps = { className?: string };
type DungeonProps = ArtProps & { dungeon?: string };
const atmosphere = {
  crypt: { light: '#7ed9ee', dark: '#143849', name: 'Затопленный склеп' },
  forge: { light: '#ffad64', dark: '#51241e', name: 'Пепельная кузница' },
  void: { light: '#ba92ff', dark: '#32215b', name: 'Разлом пустоты' },
};
function themeFor(dungeon?: string) { return dungeon === 'forge' ? atmosphere.forge : dungeon === 'void' ? atmosphere.void : atmosphere.crypt; }

export function HunterArt({heroClass = '',rank = 'E',activeSet = '',className = ''}:ArtProps & {heroClass?:string;rank?:string;activeSet?:string}) {
  return <div className={`aura-hunter-art ${className}`} data-class={heroClass} data-rank={rank} data-set={activeSet} role="img" aria-label="Охотник Системы с клинком ауры">
    <span className="aura-character-portrait aura-portrait-hunter"/><span className="aura-character-vignette"/>
    <span className="aura-hunter-sigil" aria-hidden="true"/><span className="aura-hunter-sparks" aria-hidden="true"/>
  </div>;
}

export function EnemyArt({dungeon = 'crypt',boss = true,event,className = ''}:DungeonProps & {boss?:boolean;event?:string}) {
  const theme = themeFor(dungeon);
  return <div className={`aura-enemy-art aura-enemy-${dungeon} ${boss ? 'is-boss' : 'is-guard'} ${className}`} role="img" aria-label={`${boss ? 'Владыка' : 'Страж'}: ${theme.name}`}>
    <span className={`aura-character-portrait ${event==='false_gate'?'aura-portrait-warden':`aura-portrait-${dungeon === 'forge' ? 'forge' : dungeon === 'void' ? 'void' : 'crypt'}`}`}/><span className="aura-character-vignette"/>
  </div>;
}

export function DungeonArt({dungeon = 'crypt',className = ''}:DungeonProps) {
  const id = useId().replaceAll(':','');
  const {light,dark,name} = themeFor(dungeon);
  return <svg className={`aura-dungeon-art aura-dungeon-${dungeon} ${className}`} viewBox="0 0 560 300" fill="none" role="img" aria-label={name}>
    <defs>
      <radialGradient id={`${id}-sky`}><stop stopColor={dark}/><stop offset="1" stopColor="#0a0c19"/></radialGradient>
      <radialGradient id={`${id}-gate`}><stop stopColor={light} stopOpacity=".8"/><stop offset=".3" stopColor={light} stopOpacity=".22"/><stop offset="1" stopColor={dark} stopOpacity=".1"/></radialGradient>
      <linearGradient id={`${id}-floor`} x1="280" y1="180" x2="280" y2="300" gradientUnits="userSpaceOnUse"><stop stopColor={light} stopOpacity=".27"/><stop offset="1" stopColor={light} stopOpacity="0"/></linearGradient>
      <linearGradient id={`${id}-stone`}><stop stopColor="#151c30"/><stop offset=".48" stopColor="#344055"/><stop offset="1" stopColor="#151a2a"/></linearGradient>
    </defs>
    <path fill={`url(#${id}-sky)`} d="M0 0h560v300H0z"/>
    <circle cx="280" cy="135" r="173" fill={`url(#${id}-gate)`}/>
    {dungeon === 'forge' ? <>
      <path d="M0 199 36 155 48 88 67 120 87 166 111 182 142 211v89H0Zm560 0-36-44-12-67-19 32-20 46-24 16-31 29v89h142Z" fill="#281e29"/>
      <path d="m31 300 65-82 27 9 13-29 26 102m256 0 19-59 23-25 25 21 52 63" fill="#8e402d" opacity=".55"/>
      <path d="m52 294 49-62 17 13 20-33 7 58m290 25 22-56 22 17 13 36" stroke={light} strokeWidth="2" opacity=".65"/>
    </> : <>
      {[35,115,420,500].map((x,i)=><g key={x} opacity={i%2 ? '.55' : '.32'}><path d={`M${x} 257V${i%2 ? 72 : 102}h25v155Z`} fill="#202b40"/><path d={`m${x-8} ${i%2 ? 72 : 102} 20-32 21 32Zm3 8h35m-30 80h25`} stroke="#34445b" strokeWidth="4"/></g>)}
      <path d="M0 254 85 240l53 8 64-18h156l64 18 53-8 85 14v46H0Z" fill="#101827"/>
    </>}
    <path d="m135 300 101-101h88l101 101Z" fill={`url(#${id}-floor)`}/>
    {[225,249,281].map(y=><path key={y} d={`M${220-(y-210)*1.2} ${y}H${340+(y-210)*1.2}`} stroke={light} opacity=".12"/>)}
    <path d="M188 214V111a92 92 0 0 1 184 0v103h-26V111a66 66 0 0 0-132 0v103Z" fill={`url(#${id}-stone)`}/>
    <path d="M205 208v-95a75 75 0 0 1 150 0v95" stroke={light} strokeWidth="2" opacity=".8"/>
    <path d="M217 210v-96a63 63 0 0 1 126 0v96Z" fill={`url(#${id}-gate)`}/>
    <path d="M233 203v-85a47 47 0 0 1 94 0v85" stroke={light} strokeWidth="1" opacity=".23"/>
    <ellipse cx="280" cy="212" rx="85" ry="11" fill={light} opacity=".12"/>
    {[0,30,60,90,120,150,180].map(angle=><path key={angle} d="M280 28v11" stroke={light} strokeWidth="3" transform={`rotate(${angle-90} 280 111)`} opacity=".5"/>)}
    {dungeon === 'void' ? <><circle cx="280" cy="128" r="37" stroke={light} opacity=".5"/><path d="m280 82 39 67h-78Zm0 88-39-67h78Z" stroke={light} opacity=".65"/><circle cx="280" cy="127" r="9" fill={light}/></> : <path d="m280 81 20 38-20 41-20-41Z" stroke={light} strokeWidth="2" opacity=".8"/>}
    {[0,1,2,3,4,5,6,7,8].map(i=><circle key={i} cx={55+i*57} cy={44+(i*37)%150} r={i%3+0.6} fill={light} opacity={.18+i%3*.15}/>)}
  </svg>;
}

// Painted relic sheets. Each sheet is a 3x2 grid of six square cells drawn on
// pure black with wide margins, so a cell cuts out cleanly: at this size the
// generator keeps every item inside its own cell, which the 4x3 sheets did not.
// A relic not listed here falls back to the procedural SVG below.
const sheet1 = ['rusty_blade','moonlit_saber','dragon_fang','world_ender','troll_heart','world_shell'];
const sheet2 = ['apprentice_staff','hunters_bow','twin_daggers','storm_spear','cloth_robe','leather_vest'];
const sheet3 = ['chain_mail','guardian_plate','frost_aegis','astral_shroud','eternal_bulwark','copper_ring'];
const sheet4 = ['origin_crown','hourglass','echo_blade','executioner','void_reaver'];
const sheet5 = ['pilgrim_sword','ash_wand','bone_rapier','copper_glaive','thread_mantle','moss_wrap'];
const sheet6 = ['chipped_bead','lantern_shard','pilgrim_hourglass','lucky_charm','scholar_pendant','blood_amber'];

const SHEETS = [sheet1, sheet2, sheet3, sheet4, sheet5, sheet6];
const SHEET_COLS = 3;
const SHEET_ROWS = 2;

export function RelicArt({relicKey = '',rank = 'E',slot = 'accessory',level = 1,className = ''}:ArtProps & {relicKey?:string;rank?:string;slot?:string;level?:number}) {
  const id = useId().replaceAll(':','');
  const color = ({S:'#edc27a',SS:'#f0a6c8',SSS:'#ffde9b',A:'#bd8bff',B:'#77b8f4',C:'#73d4bd'} as Record<string,string>)[rank]??'#b2b8ca';
  const weapon = slot === 'weapon' || /blade|saber|spear|dagger|bow|staff|reaver|fang|ender/.test(relicKey);
  const armor = slot === 'armor' || /robe|vest|mail|plate|aegis|shroud|bulwark|shell/.test(relicKey);
  const awakening = Math.min(5,Math.max(1,level));
  // Sheets are checked in order; the first list that holds the key wins.
  const found = SHEETS.findIndex((list) => list.indexOf(relicKey) >= 0);
  const sheet = found + 1;
  const index = found >= 0 ? (SHEETS[found] ?? []).indexOf(relicKey) : -1;
  const col = index % SHEET_COLS;
  const row = Math.floor(index / SHEET_COLS);
  const x = (col / (SHEET_COLS - 1)) * 100;
  const y = (row / (SHEET_ROWS - 1)) * 100;
  if (sheet > 0) return <div className={`aura-relic-art aura-painted-relic aura-sheet-${sheet} awakening-${awakening} ${className}`} role="img" aria-label={`Реликвия, пробуждение ${level}`} style={{color}}><span className="aura-painted-relic-image" style={{backgroundPosition:`${x}% ${y}%`}}/><span className="aura-relic-runes" aria-hidden="true"/>{awakening>=4&&<span className="aura-relic-fragments" aria-hidden="true"><i/><i/><i/><i/></span>}</div>;
  return <svg className={`aura-relic-art awakening-${awakening} ${className}`} viewBox="0 0 100 100" fill="none" role="img" aria-label={weapon ? 'Реликвия оружия' : armor ? 'Реликвия брони' : 'Магический артефакт'}>
    <defs><radialGradient id={`${id}-aura`}><stop stopColor={color} stopOpacity=".27"/><stop offset="1" stopColor={color} stopOpacity="0"/></radialGradient><linearGradient id={`${id}-metal`} x1="30" y1="20" x2="70" y2="80" gradientUnits="userSpaceOnUse"><stop stopColor="#e3e7f1"/><stop offset=".43" stopColor={color}/><stop offset=".5" stopColor="#6c718e"/><stop offset="1" stopColor="#2c3049"/></linearGradient></defs>
    <circle cx="50" cy="50" r="48" fill={`url(#${id}-aura)`}/><path d="m50 7 37 21v44L50 93 13 72V28Z" stroke={color} opacity=".16"/>
    {awakening>=2&&<g stroke={color} opacity={awakening===2?'.3':'.7'} aria-hidden="true"><circle cx="50" cy="50" r="42" strokeDasharray="4 8"/><path d="m17 26 8-3-3 8m61 38-8 3 3-8m-43 24 5-6 4 7m12-76-5 6-4-7"/></g>}
    {weapon ? /bow/.test(relicKey) ? <><path d="M31 17q59 33 0 66l15-33Z" fill={`url(#${id}-metal)`}/><path d="m33 19 25 31-25 31M20 50h56m-9-7 9 7-9 7" stroke={color} strokeWidth="2"/></> : /staff|spear/.test(relicKey) ? <><path d="m34 82 27-56" stroke="#a9a4bb" strokeWidth="5"/><path d="m68 10 5 23-13 5-5-15Z" fill={`url(#${id}-metal)`}/><path d="m65 17 3 13-7 3" stroke={color} strokeWidth="2"/><path d="m35 78 8 4m-5-11 8 4" stroke={color} strokeWidth="2"/></> : <g transform="rotate(34 50 50)"><path d="m50 8 9 17-5 38h-8l-5-38Z" fill={`url(#${id}-metal)`} stroke={color} strokeWidth=".6"/><path d="M50 14v45" stroke="#f1edff" opacity=".65"/><path d="m35 59 15 5 15-5-2 8H37Z" fill={color}/><path d="M46 68h8v17h-8Z" fill="#3c3b56" stroke={color}/><path d="m50 83 5 5-5 5-5-5Z" fill={color}/>{/twin/.test(relicKey)&&<path d="m33 20 4 9-4 27-4-27Z" fill={color}/>}</g> : armor ? <><path d="m36 18 14 7 14-7 17 15-10 18-6-4 3 30-18 10-18-10 3-30-6 4-10-18Z" fill={`url(#${id}-metal)`} stroke={color}/><path d="m36 18 14 23 14-23M35 47l15 11 15-11m-31 20 16 8 16-8M50 41v34" stroke={color} strokeWidth="2"/><path d="m50 43 8 9-8 10-8-10Z" fill={color}/></> : /hourglass/.test(relicKey) ? <><path d="M32 19h36M32 81h36" stroke={color} strokeWidth="5"/><path d="M37 23c0 17 6 20 13 27-7 7-13 10-13 27h26c0-17-6-20-13-27 7-7 13-10 13-27Z" fill={`url(#${id}-metal)`} stroke={color}/><path d="m42 30 8 13 8-13Zm-3 43 11-14 11 14Z" fill={color}/></> : /heart/.test(relicKey) ? <><path d="M50 82 23 53C1 27 36 10 50 33c14-23 49-6 27 20Z" fill={`url(#${id}-metal)`} stroke={color}/><path d="m26 49 13 1 7-15 9 29 8-15h13" stroke={color} strokeWidth="3"/></> : /crown/.test(relicKey) ? <><path d="m20 33 14 12 16-25 16 25 14-12-8 40H28Z" fill={`url(#${id}-metal)`} stroke={color} strokeWidth="2"/><path d="M30 62h40" stroke={color}/><path d="m50 43 7 9-7 9-7-9Z" fill={color}/></> : <><path d="M31 28c0 31 38 31 38 0" stroke={color} strokeWidth="3"/><path d="m50 40 19 20-19 26-19-26Z" fill={`url(#${id}-metal)`} stroke={color}/><path d="m50 47 11 14-11 17-11-17Z" fill={color} opacity=".6"/><path d="M50 50v23" stroke="#fff" opacity=".5"/></>}
    {['S','SS','SSS'].includes(rank)&&<><path d="m17 14 2 5 5 2-5 2-2 5-2-5-5-2 5-2Zm66 57 2 5 5 2-5 2-2 5-2-5-5-2 5-2Z" fill={color}/><circle cx="78" cy="19" r="2" fill={color}/></>}
    {awakening>=4&&<g className="aura-relic-fragments" fill={color} aria-hidden="true"><path d="m12 40 4 6-4 7-4-7Zm76 10 4 6-4 7-4-7ZM31 8l4 6-4 5-4-5Zm37 75 4 6-4 5-4-5Z"/></g>}
  </svg>;
}
