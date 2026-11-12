import { useCallback, useEffect, useId, useRef, useState, type CSSProperties } from 'react';
import type { GameResult } from './api';
import { CropArt, Glyph } from './art';
import { mutationNames } from './character';
import { gameNumber } from './gameDisplay';
import type { FarmSound } from './sounds';
import { gameRevealPlan } from './opening';

export function GameReveal({result,name,play,onClose}:{result:GameResult;name:string;play:(sound:FarmSound)=>void;onClose:()=>void}) {
  const ref=useRef<HTMLDialogElement>(null);
  const finished=useRef(false);
  const [revealed,setRevealed]=useState(false);
  const [phase,setPhase]=useState(0);
  const timers=useRef<number[]>([]);
  const title=useId();
  const won=result.status==='won';
  const finish=useCallback(()=>{if(finished.current)return;finished.current=true;timers.current.forEach(window.clearTimeout);setRevealed(true);play(won?'legendary':'pack')},[play,won]);
  useEffect(()=>{
    ref.current?.showModal();const plan=gameRevealPlan(result.kind);
    timers.current=plan.beats.map((at,i)=>window.setTimeout(()=>{setPhase(i+1);play(`hit${i+1}` as FarmSound)},at));
    timers.current.push(window.setTimeout(finish,plan.revealAt));
    return()=>{timers.current.forEach(window.clearTimeout);ref.current?.close()};
  },[finish,play,result.kind]);
  return <dialog ref={ref} className={`farm-game-reveal ${result.kind==='upgrade'?'is-upgrade':''} phase-${phase} ${revealed?'revealed':'revealing'} ${won?'won':'lost'}`} style={{'--charge':phase} as CSSProperties} aria-labelledby={title} onCancel={e=>{e.preventDefault();onClose()}} onClick={e=>{if(e.target===e.currentTarget)onClose()}}>
    <button className="farm-icon-button farm-game-reveal-close" aria-label="Закрыть результат" onClick={onClose}><Glyph name="close"/></button>
    <span className="farm-game-label">{result.kind==='flip'?'КОИНФЛИП':'АПГРЕЙД РАСТЕНИЯ'}</span>
    <div className="farm-game-reveal-visual"><i/>{result.kind==='upgrade'&&<><svg className="farm-upgrade-rune" viewBox="0 0 200 200" aria-hidden="true"><path d="m100 9 79 46v90l-79 46-79-46V55Z"/><path d="m100 29 62 36v70l-62 36-62-36V65ZM54 100h92M100 54v92"/><circle cx="100" cy="100" r="57" strokeDasharray="2 9"/></svg><span className="farm-upgrade-scan"/>{!revealed&&<span key={phase} className="farm-upgrade-shock"/>}</>}{result.kind==='flip'?<div className="farm-reveal-coin"><Glyph name={revealed&&result.side==='tails'?'spark':'sun'} size={88}/></div>:<CropArt seed={result.seed} mutation={revealed&&won?result.mutation:'none'}/>}<div className="farm-reveal-splinters" aria-hidden="true">{Array.from({length:8},(_,i)=><i key={i} style={{'--shard':i} as CSSProperties}/>)}</div></div>
    <div className="farm-game-reveal-copy" aria-live="polite"><h2 id={title}>{revealed?won?'Получилось!':'В этот раз не вышло':result.kind==='upgrade'?['Подключаем энергию…','Собираем мутацию…','Усиливаем связь…','Проверяем совместимость…','Стабилизируем…','Раскрываем результат…'][phase]:'Монета вращается…'}</h2><p>{revealed?result.kind==='flip'?`${result.side==='heads'?'Выпало солнце':'Выпала луна'}. ${won?`Зачислено ${gameNumber(result.payout)} монет.`:'Ставка израсходована.'}`:won?`${mutationNames[result.mutation??'none']} · ${name}. Растение ждёт в амбаре.`:'Один экземпляр и ставка израсходованы.':result.kind==='upgrade'?`Шанс попытки ${((result.chance_bp??0)/100).toFixed(2)}%. Результат уже сохранён.`:'Раунд уже рассчитан и сохранён.'}</p></div>
    <button className="farm-primary full" onClick={revealed?onClose:finish}>{revealed?'Понятно':'Показать результат'}<Glyph name="forward" size={16}/></button>
  </dialog>;
}
