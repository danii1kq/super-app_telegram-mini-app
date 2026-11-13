import { useEffect, useRef, useState } from 'react';
import { Icon } from '@/shared/ui';
import type { AuraResult } from './api';
import { RelicArt } from './art';
import { lootRevealDelay, materialLabel } from './display';

const rankOrder=['E','D','C','B','A','S','SS','SSS'];
export function LootOpening({result,onClose,onInspect,play}:{result:AuraResult;onClose:()=>void;onInspect:(key:string)=>void;play:(kind:'loot'|'tap')=>void}) {
  const relics=[...(result.relic?[result.relic]:[]),...(result.bonus_relics??[])];
  const featured=[...relics].sort((a,b)=>rankOrder.indexOf(b.rank)-rankOrder.indexOf(a.rank))[0];
  const [phase,setPhase]=useState<'sealed'|'silhouette'|'reveal'|'summary'>(featured&&result.finished?'sealed':'summary');
  const sounded=useRef(false);
  useEffect(()=>{
    if(phase!=='silhouette'&&phase!=='reveal')return;
    const reduced=window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const timer=window.setTimeout(()=>setPhase(phase==='silhouette'?'reveal':'summary'),reduced?0:phase==='silhouette'?lootRevealDelay(featured?.rank??'E'):900);
    return()=>window.clearTimeout(timer);
  },[phase,featured?.rank]);
  useEffect(()=>{if(phase==='reveal'&&!sounded.current){sounded.current=true;play('loot')}},[phase,play]);
  if(phase!=='summary'&&featured)return <div className={`aura-chest-opening rank-${featured.rank} phase-${phase}`}>
    <span className="aura-overline">ДОБЫЧА УЖЕ СОХРАНЕНА В ИНВЕНТАРЕ</span><div className="aura-chest-scene"><span className="aura-chest-rays"/>{phase==='sealed'?<svg className="aura-chest-art" viewBox="0 0 220 180" fill="none" aria-hidden="true"><path d="M29 87V59c0-30 162-30 162 0v28Z" fill="#31273f" stroke="#bfa6dc" strokeWidth="2"/><path d="m22 86 88-14 88 14v66l-88 18-88-18Z" fill="#1a1724" stroke="#a68dbd" strokeWidth="2"/><path d="m22 86 88 17 88-17M110 103v67M49 46v111m122-111v111" stroke="#d0ad79" strokeWidth="5"/><path d="m110 83 16 20-16 20-16-20Z" fill="#b9a0ff"/><path d="M110 94v16" stroke="#f5ecff" strokeWidth="3"/><path d="m77 31 33-5 33 5" stroke="#d2bddf" strokeWidth="2"/></svg>:<RelicArt relicKey={featured.key} slot={featured.slot} rank={featured.rank} level={featured.level||1}/>}</div>
    <span className="aura-chest-rank">{phase==='reveal'?featured.rank:'✧'}</span><h2>{phase==='sealed'?'Сундук поверженного босса':phase==='silhouette'?'Печать раскрывается…':featured.name}</h2><p>{phase==='sealed'?`Внутри ${relics.length===1?'реликвия':`реликвий: ${relics.length}`}. Нажми, чтобы снять печать.`:phase==='silhouette'?'Система определяет след оставшейся силы.':`Реликвия ранга ${featured.rank}`}</p>
    {phase==='sealed'?<button className="aura-primary" onClick={()=>{play('tap');setPhase('silhouette')}}>Открыть сундук<Icon name="auto_awesome" size={19}/></button>:<button className="aura-secondary" onClick={()=>setPhase('summary')}>Показать всю добычу<Icon name="skip_next" size={18}/></button>}<button className="aura-text-button" onClick={()=>setPhase('summary')}>Пропустить анимацию</button>
  </div>;
  return <div className={`aura-loot aura-loot-summary ${result.won===false?'defeat':''}`}><span className="aura-overline">{result.won===false?'СИСТЕМА СОХРАНИЛА ТВОЙ ПРОГРЕСС':'ИСПЫТАНИЕ ЗАВЕРШЕНО'}</span><h2>{result.won===false?'Ты вернёшься сильнее':relics.length?'Твоя добыча':'Ещё один шаг к пробуждению'}</h2><p>{result.message}</p>
    {relics.length>0&&<div className="aura-loot-items">{relics.map((relic,index)=><button key={`${relic.key}-${index}`} className={`rank-${relic.rank}`} onClick={()=>onInspect(relic.key)}><RelicArt relicKey={relic.key} slot={relic.slot} rank={relic.rank} level={relic.level||1}/><span><small>РАНГ {relic.rank}{relic.source==='trophy'?' · ТРОФЕЙ':''}</small><strong>{relic.name}</strong><small>Осмотреть реликвию</small></span><Icon name="chevron_right" size={18}/></button>)}</div>}
    <div className="aura-loot-rewards"><span><strong>+{result.aura_gain.toLocaleString('ru-RU')}</strong>АУРА</span><span><strong>+{result.xp_gain.toLocaleString('ru-RU')}</strong>ОПЫТ</span>{result.level_up>0&&<span><strong>+{result.level_up}</strong>УРОВЕНЬ</span>}</div><div className="aura-materials">{Object.entries(result.materials_gain??{}).map(([key,count])=><span key={key}>{materialLabel(key)} ×{count}</span>)}</div><button className="aura-primary" onClick={onClose}>Продолжить историю<Icon name="arrow_forward" size={18}/></button>
  </div>;
}
