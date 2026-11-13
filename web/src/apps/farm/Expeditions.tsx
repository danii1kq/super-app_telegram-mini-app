import type { Crop, Expedition, ExpeditionRoute, FarmAction } from './api';
import { countdown } from './api';
import { CropArt, Glyph } from './art';

function RouteArt({route}:{route:string}) {
  return <svg className={`farm-route-art ${route}`} viewBox="0 0 360 200" preserveAspectRatio="xMidYMid slice" fill="none" aria-hidden="true">
    <rect width="360" height="200" rx="18" fill="currentColor" opacity=".09"/>
    <circle cx="284" cy="50" r="22" fill="currentColor" opacity=".18"/>
    <path d="M0 151Q74 65 162 127T360 117V200H0Z" fill="currentColor" opacity=".11"/>
    <path d="M0 179Q88 116 197 160T360 141V200H0Z" fill="currentColor" opacity=".17"/>
    <path d="M180 200c-67-34 91-51 24-79" stroke="currentColor" strokeWidth="15" opacity=".13"/>
    {route==='meadow' && <>
      {[55,94,267,311].map((x,i)=><g key={x} transform={`translate(${x} ${126+i%2*18})`}><path d="M0 17V-14" stroke="#6b9e68" strokeWidth="3"/><path d="M0 9c-14 2-18-6-16-12C-4-4 0 1 0 9Z" fill="#87b679"/><circle cy="-18" r="13" fill="#c3a7e3"/><circle cy="-18" r="5" fill="#f5dc91"/></g>)}
      <path d="m162 86 18-19 20 19v38h-38Z" fill="#eee3c2"/><path d="m155 87 25-27 26 27" stroke="#8c9f73" strokeWidth="7" strokeLinejoin="round"/><path d="M174 124v-21h13v21" fill="#a3b88b"/>
    </>}
    {route==='orchard' && <>{[63,125,268,319].map((x,i)=><g key={x} transform={`translate(${x} ${107+i%2*16})`}><path d="M0 52V0" stroke="#9e7955" strokeWidth="7"/><path d="M0 31-12 16m12 5 12-14" stroke="#9e7955" strokeWidth="4"/><ellipse cy="-2" rx="26" ry="32" fill={i%2?'#8aaa6d':'#709c73'}/><circle cx="-11" cy="-6" r="5" fill="#edb963"/><circle cx="9" cy="8" r="5" fill="#edb963"/></g>)}<path d="M154 154h56l-8 24h-39Z" fill="#c69b64"/><path d="M168 152c0-25 27-25 27 0" stroke="#c69b64" strokeWidth="4"/></>}
    {route==='astral' && <><path d="m24 160 66-100 62 100m72 0 47-78 61 78" fill="#9296bf" opacity=".5"/><ellipse cx="185" cy="165" rx="61" ry="13" fill="#a6bce3" opacity=".5"/>{[0,72,144,216,288].map(angle=><path key={angle} d="M184 134c-23-15-18-39 0-52 18 13 24 37 0 52Z" transform={`rotate(${angle} 184 126)`} fill={angle%144?'#d4c3ee':'#eee2f7'}/>)}<circle cx="184" cy="126" r="9" fill="#f4db96"/><path d="m57 37 3 7 7 3-7 3-3 7-3-7-7-3 7-3Zm241 55 2 5 5 2-5 2-2 5-2-5-5-2 5-2Z" fill="#d7caee"/></>}
  </svg>;
}

export function Expeditions({routes,journey,crops,coins,level,now,busy,onAction}:{routes:ExpeditionRoute[];journey:Expedition|null|undefined;crops:Crop[];coins:number;level:number;now:number;busy:boolean;onAction:(action:FarmAction)=>void}) {
  const ready=!!journey && Date.parse(journey.ready_at)<=now;
  const progress=journey?Math.max(0,Math.min(100,(now-Date.parse(journey.started_at))/Math.max(1,Date.parse(journey.ready_at)-Date.parse(journey.started_at))*100)):0;
  const rewards=(seeds:Record<string,number>)=><div className="farm-expedition-rewards">{Object.entries(seeds).map(([id,count])=><span key={id}><CropArt seed={id}/><span>{crops.find(c=>c.id===id)?.name??id}<strong>×{count}</strong></span></span>)}</div>;
  return <>
    <section className={`farm-journey ${journey?'travelling':''}`}>
      <div className="farm-journey-compass"><Glyph name="explore" size={62}/><i/><i/></div>
      <div className="farm-journey-copy"><span className="farm-eyebrow">{journey?ready?'СЕМЕНА УЖЕ ЖДУТ':'В ПУТИ':'МИР ЗА КАЛИТКОЙ'}</span><h2>{journey?journey.name:'Новые семена за горизонтом'}</h2><p>{journey?'Награда закреплена при отправлении. Забрать её можно в любое время после возвращения.':'Отправь садовника за редкими семенами, пока растёт урожай. Весь поход занимает 3 часа, а награда известна заранее.'}</p>
        {journey && <><div className="farm-journey-progress" role="progressbar" aria-label="Прогресс экспедиции" aria-valuenow={Math.round(progress)} aria-valuemin={0} aria-valuemax={100}><span style={{width:`${progress}%`}}/></div>{rewards(journey.seeds)}<button className="farm-primary" disabled={busy||!ready} onClick={()=>onAction({kind:'claim_expedition'})}><Glyph name={ready?'box':'clock'} size={18}/>{ready?'Забрать находки':`Возвращение через ${countdown(Date.parse(journey.ready_at)-now)}`}<span>+{journey.xp} опыта</span></button></>}
      </div>
    </section>
    <header className="farm-section-heading spaced"><div><h2>Куда отправимся?</h2><p>Одновременно доступна одна экспедиция. Растения на грядках продолжают расти.</p></div><span className="farm-count">{routes.length} маршрута</span></header>
    <div className="farm-expedition-grid">{routes.map((route,i)=><article className={`farm-expedition-card route-${route.id}`} key={route.id}>
      <RouteArt route={route.id}/><div className="farm-expedition-copy"><span className="farm-eyebrow">МАРШРУТ {String(i+1).padStart(2,'0')} · {Math.round(route.duration_seconds/3600)} ЧАСА</span><h2>{route.name}</h2><p>{route.description}</p><span className="farm-expedition-label">ПРИВЕЗЁТ С СОБОЙ</span>{rewards(route.seeds)}<small>И ещё +{route.xp} опыта фермы{level < route.required_level ? ` · Откроется на ${route.required_level} уровне` : ''}</small><button className="farm-primary full" disabled={busy||!!journey||coins<route.price||level<route.required_level} onClick={()=>onAction({kind:'start_expedition',item:route.id})}>{journey?journey.route===route.id?'Садовник в пути':'Дождись возвращения':level<route.required_level?`С ${route.required_level} уровня`:coins<route.price?'Не хватает монет':'Отправиться'}<span className="farm-expedition-price"><Glyph name="coin" size={16}/>{route.price.toLocaleString('ru-RU')}</span></button></div>
    </article>)}</div>
  </>;
}
