import type { Crop, FarmBonuses, LevelInfo, Mutation, Pack, Upgrade } from './api';
import { Glyph } from './art';
import { MutationGuide } from './MutationGuide';
import { upcomingUnlocks } from './progression';

export function Upgrades({upgrades,level,crops,packs,seedLevels,packLevels,bonuses,mutations,coins,busy,onUpgrade}:{upgrades:Upgrade[];level:LevelInfo;crops:Crop[];packs:Pack[];seedLevels:Record<string,number>;packLevels:Record<string,number>;bonuses:FarmBonuses;mutations:Mutation[];coins:number;busy:boolean;onUpgrade:(id:string)=>void}) {
  const upcoming = upcomingUnlocks(level.level,crops,packs,upgrades,seedLevels,packLevels);
  const completed = upgrades.reduce((sum,upgrade) => sum + upgrade.level,0);
  const total = upgrades.reduce((sum,upgrade) => sum + upgrade.max_level,0);
  const percent = (value:number) => value.toLocaleString('ru-RU',{maximumFractionDigits:2});
  return <>
    <section className="farm-progression">
      <header><span className="farm-progression-emblem"><Glyph name="upgrade" size={30}/></span><div><span className="farm-eyebrow">ОТ ПЕРВОЙ ГРЯДКИ ДО ЛЕГЕНДЫ</span><h2>Твой путь до {level.max} уровня</h2><p>Новые сорта и мастерство открываются с опытом. Монеты превращаются в постоянные улучшения.</p></div><span className="farm-progression-count"><strong>{completed}<small> / {total}</small></strong><span>улучшений получено</span></span></header>
      {upcoming.length > 0 && <div className="farm-next-unlocks" aria-label="Ближайшие открытия">{upcoming.map(unlock => <div key={unlock.level}><span className="farm-unlock-level">{unlock.level}<small>уровень</small></span><div><strong>{unlock.names.slice(0,2).join(' · ')}</strong><span>{unlock.names.length > 2 ? `И ещё ${unlock.names.length - 2} открытий` : 'Новая цель для твоей фермы'}</span></div><Glyph name="forward" size={16}/></div>)}</div>}
    </section>
    <div className="farm-upgrade-note"><Glyph name="spark"/><p>Бонусы роста фиксируются при посадке, лавка действует при продаже. Первые пять уровней дают основу, дальше начинаются мастерство и дорогие эксперименты.</p></div>
    <div className="farm-upgrades-grid">{upgrades.map(upgrade => {
      const capped = upgrade.level >= upgrade.max_level;
      const locked = !capped && level.level < upgrade.required_level;
      const stage = upgrade.level < 5 ? 'Основа' : upgrade.level < 10 ? 'Мастерство' : 'Эксперт';
      return <article key={upgrade.id} className={`farm-upgrade-card ${upgrade.id} ${locked ? 'level-locked' : ''}`}>
        <span className="farm-upgrade-art"><Glyph name={upgrade.icon} size={50}/></span><span className="farm-upgrade-stage">{stage}</span><span className="farm-eyebrow">УРОВЕНЬ {upgrade.level} / {upgrade.max_level}</span><h2>{upgrade.name}</h2><p>{upgrade.description}</p>
        <div className="farm-upgrade-progress" role="progressbar" aria-label={`Прокачка: ${upgrade.name}`} aria-valuenow={upgrade.level} aria-valuemin={0} aria-valuemax={upgrade.max_level}><span style={{width:`${upgrade.level / upgrade.max_level * 100}%`}}/></div>
        <div className="farm-upgrade-benefit">{upgrade.benefit}</div>
        {!capped && <div className="farm-upgrade-price"><span>{locked ? `Нужен ${upgrade.required_level} уровень фермы` : 'Следующее улучшение'}</span><strong><Glyph name="coin" size={15}/>{upgrade.cost.toLocaleString('ru-RU')}</strong></div>}
        <button className="farm-primary full" disabled={capped || locked || coins < upgrade.cost || busy} onClick={() => onUpgrade(upgrade.id)}>{capped ? 'Максимальный уровень' : locked ? `С ${upgrade.required_level} уровня` : coins < upgrade.cost ? 'Копим монеты' : <>Улучшить<Glyph name="plus" size={15}/></>}</button>
      </article>;
    })}</div>
    <div className="farm-current-chances"><h3>Бонусы твоей фермы</h3><span>Идеальное качество<strong>{percent(bonuses.perfect_percent)}%</strong></span><span>Отборное качество<strong>{percent(bonuses.choice_percent)}%</strong></span><span>Любая мутация<strong>{percent(bonuses.mutation_percent)}%</strong></span><span>Дополнительный урожай<strong>+{percent(bonuses.yield_percent)}%</strong></span><span>Возврат семени<strong>{percent(bonuses.nursery_percent)}%</strong></span><span>Цена продажи<strong>+{percent(bonuses.market_percent)}%</strong></span><p>Бонус урожая даёт целые дополнительные растения и шанс на ещё одно. Уход добавляет бонус к качеству, урожаю или мутации. Питомник возвращает максимум одно семя с грядки.</p></div>
    <MutationGuide mutations={mutations}/>
  </>;
}
