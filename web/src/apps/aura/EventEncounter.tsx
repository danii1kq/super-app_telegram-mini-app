import { Icon } from '@/shared/ui';
import { EnemyArt } from './art';
import type { DungeonEvent } from './api';

export function EventEncounter({event,busy,onChoose}:{event:DungeonEvent;busy:boolean;onChoose:(route:string)=>void}) {
  return <section className="aura-event-encounter" aria-labelledby="aura-event-title">
    <div className="aura-event-portrait"><EnemyArt event={event.id} dungeon={event.id==='red_gate'?'forge':'void'} boss/></div>
    <div className="aura-event-copy"><span className="aura-overline"><Icon name="warning_amber" size={18}/>СИСТЕМА: ОШИБКА ОЦЕНКИ ВРАТ</span>
      <span className="aura-event-rank">{event.rank}<small>ИСТИННЫЙ РАНГ</small></span>
      <h2 id="aura-event-title">{event.name}</h2><p>{event.description}</p>
      <div className="aura-event-warning"><Icon name="shield" size={22}/><p>{event.warning}</p></div>
      <p className="aura-event-footnote">Уже найденные материалы сохранятся. Уникальная добыча доступна только после победы. Сложность примеров останется выбранной тобой.</p>
      <div className="aura-event-actions"><button className="aura-primary" disabled={busy} onClick={()=>onChoose('event_enter')}>Принять вызов<Icon name="arrow_forward" size={18}/></button><button className="aura-secondary" disabled={busy} onClick={()=>onChoose('event_leave')}>Отступить с материалами<Icon name="logout" size={18}/></button></div>
    </div>
  </section>;
}

export function AnomalyGuide() {
  return <section className="aura-anomaly-guide"><span className="aura-overline">ЗАКРЫТЫЙ АРХИВ СИСТЕМЫ</span><h2>Не каждые врата говорят правду</h2><p>В редком походе после зачистки комнаты может открыться аномалия. Угроза и условия будут показаны до принятия вызова. Высшие реликвии нельзя изготовить: их получают за победу в опасных событиях.</p><div className="aura-anomaly-types">
    <article><strong>Ложная оценка · S</strong><p>За обычными вратами скрывается Смотритель безмолвия. Усиленный босс меняет фазы боя и наказывает ошибки.</p></article>
    <article><strong>Красные врата · A</strong><p>Цепь усиленных противников. После принятия потребуется закончить испытание или потерпеть поражение.</p></article>
    <article><strong>Скрытое святилище · B</strong><p>Забытый хранитель предлагает испытание за реликвию из отдельной коллекции.</p></article>
  </div></section>;
}
