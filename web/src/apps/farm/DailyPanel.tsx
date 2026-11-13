import type { Crop, DailyInfo } from './api';
import { Glyph } from './art';

/**
 * DailyPanel is the seven day login streak. It shows the whole cycle so the
 * seventh day is visible from day one, which is the point of a streak: the
 * player should know what they are being kept for.
 */
export function DailyPanel({daily, crops, busy, onClaim}:{daily:DailyInfo; crops:Crop[]; busy:boolean; onClaim:()=>void}) {
  const today = daily.rewards.find(item => item.day === daily.day);
  const seedNames = (seeds: Record<string, number>) =>
    Object.entries(seeds).map(([id, count]) => `${crops.find(crop => crop.id === id)?.name ?? id} ×${count}`).join(', ');
  return <section className="farm-daily">
    <header className="farm-section-heading"><div><h2><Glyph name="spark" size={16}/>Ежедневный вход</h2><p>{daily.streak > 0 ? `Серия: ${daily.streak} ${daily.streak === 1 ? 'день' : 'дней'}. Лучшая: ${daily.best}.` : 'Заходи каждый день, награда растёт.'}</p></div></header>
    <div className="farm-daily-track">{daily.rewards.map(reward => <span key={reward.day} className={`farm-daily-day ${reward.day === daily.day ? 'today' : ''} ${reward.day < daily.day ? 'past' : ''}`} title={seedNames(reward.seeds) || `${reward.coins} монет`}>
      <small>{reward.day}</small>
      <Glyph name="coin" size={12}/>
      <b>{reward.coins}</b>
    </span>)}</div>
    <button className="farm-primary full" disabled={busy || !daily.claimable} onClick={onClaim}>
      {daily.claimable ? <><Glyph name="spark" size={14}/>Забрать день {daily.day}{today ? <span className="farm-daily-pay">{today.coins}</span> : null}</> : 'Сегодня уже получено'}
    </button>
  </section>;
}
