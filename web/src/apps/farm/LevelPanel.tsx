import type { LevelInfo } from './api';
import { Glyph } from './art';

/**
 * LevelPanel shows the current level, the XP bar and any unclaimed rewards.
 *
 * The bar is server-authoritative: `into` and `need` come from the API so the
 * curve lives in one place. At the cap the bar is replaced by a finished state
 * instead of a full track that never moves.
 */
export function LevelPanel({level, busy, onClaim}:{level:LevelInfo; busy:boolean; onClaim:(level:number)=>void}) {
  const capped = level.level >= level.max;
  const pending = level.pending[0];
  const percent = capped ? 100 : Math.min(100, Math.round(level.into / Math.max(1, level.need) * 100));
  return <section className="farm-level">
    <span className="farm-level-badge">{level.level}</span>
    <div>
      <strong>{level.title}</strong>
      <span>{capped ? 'Высший уровень достигнут' : `${level.into} / ${level.need} опыта до уровня ${level.level + 1}`}</span>
      <span className="farm-level-track"><span style={{width:`${percent}%`}}/></span>
    </div>
    {pending !== undefined && <button className="farm-level-claim" disabled={busy} onClick={()=>onClaim(pending)}>
      <Glyph name="spark" size={13}/>Награда<span>ур. {pending}</span>
    </button>}
  </section>;
}
