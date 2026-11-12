import { useCallback, useEffect, useId, useRef, useState } from 'react';
import type { GameResult } from './api';
import { CropArt, Glyph } from './art';
import { mutationNames } from './character';
import { gameNumber } from './gameDisplay';
import type { FarmSound } from './sounds';

export function GameReveal({result,name,play,onClose}:{result:GameResult;name:string;play:(sound:FarmSound)=>void;onClose:()=>void}) {
  const ref=useRef<HTMLDialogElement>(null);
  const finished=useRef(false);
  const [revealed,setRevealed]=useState(false);
  const title=useId();
  const won=result.status==='won';
  const finish=useCallback(()=>{if(finished.current)return;finished.current=true;setRevealed(true);play(won?'legendary':'pack')},[play,won]);
  useEffect(()=>{ref.current?.showModal();const timer=window.setTimeout(finish,1800);return()=>{window.clearTimeout(timer);ref.current?.close()}},[finish]);
  return <dialog ref={ref} className={`farm-game-reveal ${revealed?'revealed':'revealing'} ${won?'won':'lost'}`} aria-labelledby={title} onCancel={e=>{e.preventDefault();onClose()}} onClick={e=>{if(e.target===e.currentTarget)onClose()}}>
    <button className="farm-icon-button farm-game-reveal-close" aria-label="Закрыть результат" onClick={onClose}><Glyph name="close"/></button>
    <span className="farm-game-label">{result.kind==='flip'?'КОИНФЛИП':'АПГРЕЙД РАСТЕНИЯ'}</span>
    <div className="farm-game-reveal-visual"><i/>{result.kind==='flip'?<div className="farm-reveal-coin"><Glyph name={revealed&&result.side==='tails'?'spark':'sun'} size={88}/></div>:<CropArt seed={result.seed} mutation={revealed&&won?result.mutation:'none'}/>}</div>
    <div className="farm-game-reveal-copy" aria-live="polite"><h2 id={title}>{revealed?won?'Получилось!':'В этот раз не вышло':'Смотрим результат…'}</h2><p>{revealed?result.kind==='flip'?`${result.side==='heads'?'Выпало солнце':'Выпала луна'}. ${won?`Зачислено ${gameNumber(result.payout)} монет.`:'Ставка израсходована.'}`:won?`${mutationNames[result.mutation??'none']} · ${name}. Растение ждёт в амбаре.`:'Один экземпляр и ставка израсходованы.':'Раунд уже рассчитан и сохранён.'}</p></div>
    <button className="farm-primary full" onClick={revealed?onClose:finish}>{revealed?'Понятно':'Показать результат'}<Glyph name="forward" size={16}/></button>
  </dialog>;
}
