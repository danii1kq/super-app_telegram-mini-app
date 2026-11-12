import { CropArt, Glyph } from './art';
import { plantCharacter, plantWeight } from './character';
import type { Crop, FarmAction, Trophy } from './api';

const mutations:Record<string,string>={none:'Обычное',gold:'Золотое',crystal:'Кристальное',giant:'Гигантское'};
const qualities:Record<string,string>={normal:'Обычное',choice:'Отборное',perfect:'Идеальное'};

export function Showcase({trophies,crops,busy,onAction,onGarden}:{trophies:Trophy[];crops:Crop[];busy:boolean;onAction:(action:FarmAction)=>void;onGarden:()=>void}){
  return <><div className="farm-showcase-note"><Glyph name="trophy" size={27}/><div><h2>Твои шесть особенных находок</h2><p>Выбирай экземпляры из амбара. На витрине они сохраняются навсегда, пока ты не решишь вернуть их обратно.</p></div><span className="farm-count">{trophies.length}/6</span></div><div className="farm-trophy-grid">{Array.from({length:6},(_,i)=>{
    const trophy=trophies[i];if(!trophy)return <button key={`empty-${i}`} className="farm-trophy-empty" onClick={onGarden}><span><Glyph name="plus" size={27}/></span><strong>Место для истории</strong><small>Сохрани растение из амбара</small></button>;
    const item=trophy.plant;
    return <article key={item.id} className={`farm-trophy mutation-${item.mutation}`}><span className="farm-trophy-number">ЭКЗЕМПЛЯР {String(i+1).padStart(2,'0')}</span><div className="farm-trophy-dome"><CropArt seed={item.seed} mutation={item.mutation}/><span className="farm-trophy-plinth"/></div><h3>{crops.find(c=>c.id===item.seed)?.name ?? item.seed}</h3><span className="farm-trophy-kind">{mutations[item.mutation]} · {qualities[item.quality]}</span><p>{plantCharacter(item.seed,item.mutation).label}</p><div className="farm-trophy-stats"><span><Glyph name="seed" size={14}/>{plantWeight(item.weight)}</span><span><Glyph name="coin" size={14}/>{item.value.toLocaleString('ru-RU')}</span></div><small className="farm-trophy-date">Сохранён {new Date(trophy.saved_at).toLocaleDateString('ru-RU')}</small><button className="farm-secondary full" disabled={busy} onClick={()=>onAction({kind:'return_trophy',item:item.id})}>Вернуть в амбар<Glyph name="box" size={14}/></button></article>;
  })}</div></>;
}
