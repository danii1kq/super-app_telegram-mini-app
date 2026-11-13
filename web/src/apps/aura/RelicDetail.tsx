import { useState } from 'react';
import { Icon } from '@/shared/ui';
import type { AuraAction, AuraPlayer, AuraRelic } from './api';
import { RelicArt } from './art';
import { canCraft, materialLabel, relicComparison } from './display';

const slotNames:Record<string,string>={weapon:'Оружие',armor:'Броня',accessory:'Артефакт'};
const sourceNames:Record<string,string>={dungeon:'Добыча врат',false_gate:'Ложные врата S',red_gate:'Красные врата',hidden_shrine:'Скрытое святилище',trophy:'Трофей первой победы',condition:'Особое условие зачистки'};
const dungeonNames:Record<string,string>={crypt:'Затопленный склеп',forge:'Пепельная кузница',void:'Разлом пустоты'};
const number=(value:number)=>value.toLocaleString('ru-RU',{maximumFractionDigits:2});

function Cost({aura,materials}:{aura:number;materials:Record<string,number>}) {
  return <div className="aura-detail-cost"><span>✦ {number(aura)} ауры</span>{Object.entries(materials).map(([key,count])=><span key={key}><Icon name="diamond" size={14}/>{materialLabel(key)} ×{number(count)}</span>)}</div>;
}

export function RelicDetail({relic,owned,equipped,player,busy,favorites,onAction}:{relic:AuraRelic;owned:boolean;equipped?:AuraRelic;player:AuraPlayer;busy:boolean;favorites:number;onAction:(action:AuraAction,body:Record<string,unknown>)=>void}) {
  const [preview,setPreview]=useState(false);
  const level=owned?relic.level:1;
  const maxed=level>=5;
  const visualLevel=preview?Math.min(5,level+1):level;
  const difference=relicComparison(relic,equipped);
  const valueUnit=relic.passive==='time'?'с':'п.п.';
  const displayDifference=relic.passive==='time'&&difference!==null?difference/100:difference;
  const upgradeAffordable=canCraft(player.aura,player.materials??{},relic.upgrade_cost??0,relic.upgrade_materials??{});
  const craftAffordable=canCraft(player.aura,player.materials??{},relic.craft_aura??0,relic.craft_cost??{});
  const locked=busy||!!player.session;
  return <div className={`aura-relic-detail rank-${relic.rank}`}>
    <div className="aura-detail-art"><span className="aura-rank-tag">{relic.rank}</span><RelicArt relicKey={relic.key} slot={relic.slot} rank={relic.rank} level={visualLevel}/><span className="aura-detail-art-label">{owned?`ПРОБУЖДЕНИЕ ${visualLevel} / 5`:'АРХИВ РЕЛИКВИЙ'}</span>{owned&&!maxed&&<div className="aura-preview-tabs" aria-label="Внешний вид пробуждения"><button aria-pressed={!preview} className={!preview?'active':''} onClick={()=>setPreview(false)}>Сейчас</button><button aria-pressed={preview} className={preview?'active':''} onClick={()=>setPreview(true)}>Следующий уровень</button></div>}</div>
    <div className="aura-detail-copy"><span className="aura-overline">{slotNames[relic.slot]} · {sourceNames[relic.source]??'Реликвия Системы'}{relic.dungeon?` · ${dungeonNames[relic.dungeon]??relic.dungeon}`:''}</span><h2>{relic.name}</h2><p className="aura-detail-lore">{relic.lore||'Магическая печать хранит след пройденных врат.'}</p>
      <section className="aura-detail-effect"><span className="aura-overline">{preview?'ПОСЛЕ ПРОБУЖДЕНИЯ':'ДЕЙСТВУЮЩИЙ ЭФФЕКТ'}</span><strong>{preview?relic.next_description||relic.description:relic.description}</strong>{owned&&!maxed&&!preview&&relic.next_description&&<small>Следующий уровень: {relic.next_description}</small>}</section>
      {relic.curse&&<section className="aura-detail-curse"><Icon name="warning_amber" size={21}/><div><span>ЦЕНА ПРОКЛЯТИЯ</span><p>{relic.curse}</p></div></section>}
      {relic.set_id&&<section className="aura-detail-set"><Icon name="auto_awesome" size={20}/><div><strong>{relic.set_name}</strong><p>{relic.set_bonus}</p><small>Бонус действует при двух надетых предметах набора.</small></div></section>}
      {equipped&&equipped.key!==relic.key&&<section className="aura-detail-compare"><span className="aura-overline">В ЭТОМ СЛОТЕ: {equipped.name}</span>{displayDifference===null?<p>{equipped.description}<small>Другой эффект. Выбирай под свою сборку.</small></p>:<p>{displayDifference>0?'+':''}{number(displayDifference)} {valueUnit} к этому эффекту<small>{equipped.description}</small></p>}</section>}
      {relic.condition_hint&&<p className="aura-detail-hint"><Icon name="visibility" size={17}/>{relic.condition_hint}</p>}
      {player.session&&<p className="aura-detail-footnote">Экипировка и пробуждение доступны между походами.</p>}
      <div className="aura-detail-actions">{owned?<><button className="aura-primary" disabled={locked} onClick={()=>onAction('relics/equip',{key:relic.key,slot:relic.worn?'':relic.slot})}>{relic.worn?'Снять реликвию':'Экипировать'}<Icon name={relic.worn?'remove':'check'} size={18}/></button>{!maxed&&<><Cost aura={relic.upgrade_cost??0} materials={relic.upgrade_materials??{}}/><button className="aura-secondary" disabled={locked||!upgradeAffordable} onClick={()=>onAction('relics/upgrade',{key:relic.key})}>Пробудить до {level+1} уровня<Icon name="upgrade" size={19}/></button></>}<button className="aura-text-button aura-favorite-button" disabled={busy||!relic.favorite&&favorites>=3} onClick={()=>onAction('relics/favorite',{key:relic.key})}><Icon name={relic.favorite?'star':'star_border'} size={19}/>{relic.favorite?'Убрать с витрины':`На витрину · ${favorites}/3`}</button></>:relic.craftable?<><Cost aura={relic.craft_aura??0} materials={relic.craft_cost??{}}/><button className="aura-primary" disabled={locked||!craftAffordable} onClick={()=>onAction('relics/craft',{key:relic.key})}>Изготовить реликвию<Icon name="construction" size={18}/></button></>:<p className="aura-detail-footnote">Этот предмет нельзя изготовить. Источник: {sourceNames[relic.source]??'особая победа'}.</p>}</div>
    </div>
  </div>;
}
