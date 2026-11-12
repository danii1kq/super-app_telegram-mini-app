import { useState } from 'react';
import { CropArt, Glyph } from './art';
import { countdown, type Crop, type FarmAction, type FarmState, type Recipe } from './api';

export function Laboratory({state,recipes,crops,now,busy,onAction}: {state:FarmState;recipes:Recipe[];crops:Crop[];now:number;busy:boolean;onAction:(action:FarmAction)=>void}){
  const [sources,setSources]=useState<Record<string,string>>({});
  const job=state.breeding;
  const ready=job && Date.parse(job.ready_at)<=now;
  const sourceName=(id:string)=>crops.find(c=>c.id===id)?.name ?? id;
  const resultName=(id:string)=>recipes.find(r=>r.crop.id===id)?.crop.name ?? id;
  return <>
    <section className={`farm-lab-station ${job?'working':''}`}><div className="farm-lab-vessel" aria-hidden="true"><span/><Glyph name="spark" size={38}/><i/><i/><i/></div><div><span className="farm-eyebrow">ЛАБОРАТОРИЯ СКРЕЩИВАНИЯ</span><h2>{job?ready?'Гибрид готов к посадке':`${sourceName(job.source)} превращается…`:'У повторов появилась новая жизнь.'}</h2><p>{job?`${resultName(job.result)} · два одинаковых семени, один новый сорт`:'Два одинаковых семени и 3 часа ожидания. Шесть новых гибридов, которые можно получить только здесь.'}</p>{job?<button className="farm-primary" disabled={!ready || busy} onClick={()=>onAction({kind:'claim_breed'})}><Glyph name={ready?'check':'clock'} size={17}/>{ready?'Забрать гибрид':`До результата ${countdown(Date.parse(job.ready_at)-now)}`}</button>:<span className="farm-lab-free"><Glyph name="check" size={15}/>Лаборатория свободна</span>}</div></section>
    <header className="farm-section-heading spaced"><div><h2>Открывай новые сорта</h2><p>Можно использовать любой из двух исходных видов. Нужно 2 семени одного вида.</p></div><span className="farm-count">{state.hybrids?.length ?? 0}/6</span></header>
    <div className="farm-recipe-grid">{recipes.map(recipe=>{
      const selected=sources[recipe.crop.id] ?? recipe.sources.find(id=>(state.seeds[id]??0)>=2) ?? recipe.sources[0]!;
      const enough=(state.seeds[selected]??0)>=2;
      const discovered=state.hybrids?.includes(recipe.crop.id);
      return <article key={recipe.crop.id} className={`farm-recipe ${discovered?'discovered':''}`}><div className="farm-recipe-art"><CropArt seed={recipe.crop.id}/><span className="farm-rarity-label hybrid">ГИБРИД</span></div><h3>{recipe.crop.name}</h3><p>{discovered?'Уже получен в лаборатории':'Новый сорт ждёт своего открытия'}</p><div className="farm-recipe-sources">{recipe.sources.map(id=><button key={id} className={selected===id?'selected':''} aria-pressed={selected===id} onClick={()=>setSources(previous=>({...previous,[recipe.crop.id]:id}))}><CropArt seed={id}/><span>{sourceName(id)}<small>{state.seeds[id]??0} в запасе</small></span></button>)}</div><button className="farm-primary full" disabled={busy || Boolean(job) || !enough} onClick={()=>onAction({kind:'breed',seed:selected})}>{job?'Лаборатория занята':!enough?'Нужно 2 одинаковых семени':'Скрестить 2 семени'}<Glyph name="spark" size={16}/></button><small className="farm-recipe-time"><Glyph name="clock" size={12}/>3 часа · 1 гибридное семя</small></article>;
    })}</div>
  </>;
}
