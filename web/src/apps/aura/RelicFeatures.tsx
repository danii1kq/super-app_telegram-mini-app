import { Icon } from '@/shared/ui';
import type { AuraRelic, AuraSet, JournalEntry } from './api';
import { RelicArt } from './art';

export function RelicShowcase({items,onInspect,onBrowse}:{items:AuraRelic[];onInspect:(key:string)=>void;onBrowse:()=>void}) {
  return <section className="aura-panel aura-item-showcase"><header><div><span className="aura-overline">ТРОФЕИ ТВОЕЙ ИСТОРИИ</span><h2>Витрина охотника</h2></div><span className="aura-level">{items.length}/3</span></header><div className="aura-showcase-slots">{[0,1,2].map(index=>{
    const relic=items[index];
    return relic?<button key={index} onClick={()=>onInspect(relic.key)} aria-label={`Осмотреть трофей: ${relic.name}`}><RelicArt relicKey={relic.key} rank={relic.rank} slot={relic.slot} level={relic.level}/><strong>{relic.name}</strong><small>РАНГ {relic.rank} · УР. {relic.level}</small></button>:<button key={index} className="aura-showcase-empty" onClick={onBrowse}><Icon name="star_border" size={27}/><strong>Твой следующий трофей</strong><small>Выбери реликвию в инвентаре</small></button>;
  })}</div></section>;
}

export function RelicSets({sets,catalog,onInspect}:{sets:AuraSet[];catalog:AuraRelic[];onInspect:(key:string)=>void}) {
  if(!sets.length)return null;
  return <section className="aura-set-section"><header><div><span className="aura-overline">ДВЕ РЕЛИКВИИ, ОДНА КЛЯТВА</span><h2>Собери силу набора</h2></div></header><div className="aura-set-grid">{sets.map(set=><article className={`aura-set-card ${set.active?'active':''}`} key={set.id}><span>{set.active?'БОНУС АКТИВЕН':`НАДЕТО ${set.equipped}/${set.required}`}</span><h3>{set.name}</h3><p>{set.description}</p><div className="aura-set-pieces">{set.keys.map(key=>{
    const relic=catalog.find(item=>item.key===key);
    return relic?<button key={key} onClick={()=>onInspect(key)} aria-label={`Осмотреть часть набора: ${relic.name}`}><RelicArt relicKey={key} rank={relic.rank} slot={relic.slot} level={relic.level||1}/></button>:null;
  })}</div></article>)}</div></section>;
}

export function RelicJournal({entries,onInspect}:{entries:JournalEntry[];onInspect:(key:string)=>void}) {
  if(!entries.length)return null;
  return <section className="aura-relic-journal"><div className="aura-section-title"><div><span className="aura-overline">НЕ ВСЕ СЛЕДЫ ВИДНЫ СРАЗУ</span><h2>Журнал скрытой добычи</h2></div></div><div className="aura-journal-grid">{entries.map(entry=><article className={`aura-journal-entry ${entry.discovered?'discovered':''}`} key={entry.id}><span>{entry.discovered?'РЕЛИКВИЯ НАЙДЕНА':entry.revealed?'УСЛОВИЕ РАСКРЫТО':'НЕИЗВЕСТНЫЙ СЛЕД'}</span><h3>{entry.name}</h3><p>{entry.hint}</p>{(entry.revealed||entry.discovered)&&<button className="aura-text-button" onClick={()=>onInspect(entry.key)}>Осмотреть запись<Icon name="arrow_forward" size={16}/></button>}</article>)}</div></section>;
}
