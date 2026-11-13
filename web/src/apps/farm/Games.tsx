import { useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import type { Crop, FarmAction, FarmState, GameResult, GameRules, Mutation } from './api';
import { getCoinflips, getUpgradeQuote } from './api';
import { CropArt, Glyph } from './art';
import { mutationNames } from './character';
import { coinflipPayout, gameMultiplier, gameNumber, gameStake, multiplierText } from './gameDisplay';
import type { FarmSound } from './sounds';
import { GameReveal } from './GameReveal';

type Mode='crash'|'flip'|'upgrade';
const modes=[['crash','lamp','Переросток','Забери до взрыва'],['flip','sun','Коинфлип','Солнце или луна'],['upgrade','spark','Апгрейд','Рискни ради редкости']] as const;

export function Games({state,crops,mutations,rules,serverTime,receivedAt,me,level,seedLevels,busy,onAction,play}:{state:FarmState;crops:Crop[];mutations:Mutation[];rules:GameRules;serverTime:string;receivedAt:number;me:string;level:number;seedLevels:Record<string,number>;busy:boolean;onAction:(action:FarmAction)=>void;play:(sound:FarmSound)=>void}) {
  const [mode,setMode]=useState<Mode>('crash');
  const [stakeText,setStakeText]=useState('1000');
  const [side,setSide]=useState('heads');
  const [sourceID,setSourceID]=useState('');
  const [target,setTarget]=useState(()=>crops.filter(c=>(seedLevels[c.id]??1)<=level).sort((a,b)=>b.price-a.price)[0]?.id??'carrot');
  const [targetMutation,setTargetMutation]=useState('crystal');
  const [frame,setFrame]=useState(Date.now());
  const [reveal,setReveal]=useState<GameResult|null>(null);
  const crash=state.crash;
  const running=crash?.status==='running';
  const crashReturn=running?(crash.return_percent??rules.crash_return_percent):rules.crash_return_percent;
  const percent=(value:number)=>value.toLocaleString('ru-RU',{maximumFractionDigits:2});
  const history=state.game_history??[];
  const lastSeen=useRef(history.at(-1)?.id);
  const lastBeat=useRef('');
  const stake=gameStake(stakeText,rules.min_stake,rules.max_stake);
  const liveNow=Date.parse(serverTime)+Math.max(0,frame-receivedAt);
  const bp=crash?(running?gameMultiplier(liveNow,crash.started_at,rules.crash_growth,rules.crash_max_bp):crash.multiplier_bp):100;
  const source=state.barn.find(item=>item.id===sourceID)??state.barn[0];
  const flips=useQuery({queryKey:['farm-coinflips',me],queryFn:getCoinflips,enabled:mode==='flip',refetchInterval:2000,retry:1});
  const targetUnlocked=(seedLevels[target]??1)<=level&&!!mutations.find(m=>m.id===targetMutation&&m.unlocked);
  const quote=useQuery({queryKey:['farm-upgrade-quote',me,source?.id,source?.quantity,source?.value,target,targetMutation,stake,state.upgrades.market,state.xp,state.upgrades.genetics,rules.upgrade_return_percent,rules.upgrade_max_chance_bp],queryFn:()=>getUpgradeQuote({item:source!.id,seed:target,upgrade:targetMutation,stake:stake!}),enabled:mode==='upgrade'&&!!source&&stake!==null&&targetUnlocked,retry:false});
  useEffect(()=>{const timer=window.setInterval(()=>setFrame(Date.now()),running&&mode==='crash'?50:1000);return()=>window.clearInterval(timer)},[running,mode]);
  useEffect(()=>{
    const last=history.at(-1);if(!last||lastSeen.current===last.id)return;lastSeen.current=last.id;
    if(last.kind==='flip'||last.kind==='upgrade'){setReveal(last);return}
    play(last.status==='won'?'legendary':last.status==='cashed'?'sell':'pack');
  },[history,play]);
  useEffect(()=>{
    if(!running||!crash||mode!=='crash')return;
    const beat=Math.min(5,Math.floor(bp/100));const key=`${crash.id}:${beat}`;
    if(lastBeat.current!==key){lastBeat.current=key;play((`hit${Math.max(1,beat)}`) as FarmSound)}
  },[bp,running,crash,play,mode]);
  const valid=stake!==null&&state.coins>=stake;
  const myOpen=flips.data?.flips.find(f=>f.host_id===me&&f.status==='open');
  const open=flips.data?.flips.filter(f=>f.status==='open')??[];
  const name=(id:string)=>crops.find(c=>c.id===id)?.name??id;
  const lastUpgrade=[...history].reverse().find(item=>item.kind==='upgrade');
  return <>
    <div className="farm-games-heading"><span className="farm-game-label"><Glyph name="game" size={16}/>ИГРОВЫЕ МОНЕТЫ</span><span><Glyph name="coin" size={16}/>{gameNumber(state.coins)}</span></div>
    <div className="farm-game-modes" aria-label="Режим игры">{modes.map(([id,icon,label,caption])=><button key={id} className={mode===id?'active':''} aria-pressed={mode===id} onClick={()=>setMode(id)}><Glyph name={icon} size={25}/><span><strong>{label}</strong><small>{caption}</small></span><Glyph name="forward" size={16}/></button>)}</div>
    <div className="farm-games-layout"><section className="farm-game-main">
      {mode==='crash'&&<>
        <div className={`farm-crash-stage ${crash?.status??'idle'} tier-${Math.min(3,Math.floor(bp/200))}`}>
          <div className="farm-crash-grid"/><div className="farm-crash-orbit"/><span className="farm-game-eyebrow">{running?'РАСТЁТ':crash?.status==='busted'?'РАСТЕНИЕ ЛОПНУЛО':crash?.status==='cashed'?'УРОЖАЙ ЗАБРАН':'НАЧНИ РАЗГОН'}</span>
          <strong className="farm-crash-multiplier">{multiplierText(bp)}</strong>
          <div className="farm-crash-fruit" style={{transform:`scale(${1+Math.log(bp/100)*.12})`}}><CropArt seed="pumpkin" mutation={bp>=400?'gold':'none'}/><svg viewBox="0 0 120 120" aria-hidden="true" className="farm-crash-cracks"><path d="m61 41-9 17 12 9-11 14 6 21M32 59l15 6-7 13m48-22-16 9 8 13" stroke="currentColor" strokeWidth="2" fill="none"/></svg></div>
          <div className="farm-crash-stage-caption">{running?<>Сейчас к получению <b>{gameNumber(Math.floor(crash.stake*bp/100))}</b></>:crash?.status==='cashed'?<>Зачислено <b>{gameNumber(crash.payout)} монет</b></>:crash?.status==='busted'?<>Ставка {gameNumber(crash.stake)} израсходована</>:'Множитель растёт до ×10.00'}</div>
        </div>
        <div className="farm-game-controls">{running?<button className="farm-primary full farm-cashout" disabled={busy} onClick={()=>onAction({kind:'crash_cashout',item:crash.id})}>Забрать {gameNumber(Math.floor(crash.stake*bp/100))}<Glyph name="coin" size={21}/></button>:<button className="farm-primary full large" disabled={busy||!valid} onClick={()=>onAction({kind:'crash_start',stake:stake!})}>{busy?'Отправляем…':'Начать разгон'}<span>{stake?gameNumber(stake):'Выбери ставку'}</span></button>}<p>Выплата считается по времени сервера. На ×10 срабатывает автосбор. Закрытие страницы продолжает раунд.</p></div>
      </>}
      {mode==='flip'&&<div className="farm-flip-panel">
        <div className={`farm-flip-coin ${side}`}><Glyph name={side==='heads'?'sun':'spark'} size={67}/></div><h2>На какой ты стороне?</h2><p>Два игрока вносят равные ставки. Победитель получает банк за вычетом {rules.flip_fee_percent}% комиссии.</p>
        <div className="farm-side-pick">{([['heads','sun','Солнце'],['tails','spark','Луна']] as const).map(([id,icon,label])=><button key={id} className={side===id?'active':''} aria-pressed={side===id} onClick={()=>setSide(id)}><Glyph name={icon}/>{label}<small>50%</small></button>)}</div>
        <button className="farm-primary full" disabled={busy||!valid||!!myOpen} onClick={()=>onAction({kind:'flip_create',stake:stake!,side})}>{myOpen?'Твоя заявка уже ждёт соперника':'Создать коинфлип'}<span>{stake?gameNumber(stake):'-'}</span></button>
        <div className="farm-flip-lobby"><header><h3>Открытые заявки</h3><span>{open.length}</span></header>{flips.isError?<p role="alert">Не удалось загрузить заявки. <button onClick={()=>void flips.refetch()}>Повторить</button></p>:flips.isLoading?<p>Ищем заявки…</p>:!open.length?<p className="farm-game-empty">Пока заявок нет. Создай свою и пригласи друга на ферму.</p>:open.map(f=><article key={f.id}><span className={`farm-flip-avatar ${f.host_side}`}><Glyph name={f.host_side==='heads'?'sun':'spark'}/></span><div><strong>{f.host_name}{f.host_id===me?' · ты':''}</strong><small>{f.host_side==='heads'?'Солнце':'Луна'} · комиссия {f.fee_percent}% · {Math.max(0,Math.ceil((Date.parse(f.expires_at)-liveNow)/60000))} мин. до возврата</small></div><b>{gameNumber(f.stake)}</b><button className="farm-secondary" disabled={busy||(f.host_id!==me&&state.coins<f.stake)} onClick={()=>onAction({kind:f.host_id===me?'flip_cancel':'flip_join',item:f.id})}>{f.host_id===me?'Вернуть ставку':'Играть'}</button></article>)}</div>
        {(flips.data?.flips??[]).filter(f=>f.status==='done').slice(0,5).map(f=><div className={`farm-flip-result ${f.winner_id===me?'won':'lost'}`} key={f.id}><Glyph name={f.result_side==='heads'?'sun':'spark'} size={25}/><div><strong>{f.winner_id===me?'Ты победил':'Победил соперник'}</strong><small>{f.host_name} · {f.guest_name}</small></div><b>{f.winner_id===me?`+${gameNumber(coinflipPayout(f.stake,f.fee_percent))}`:'0'}</b></div>)}
        <small className="farm-game-note">Заявку можно отменить до принятия. Через 10 минут ожидания ставка возвращается при обновлении списка. В лобби настоящие игроки.</small>
      </div>}
      {mode==='upgrade'&&<div className="farm-upgrade-game"><span className="farm-game-label">ИЗ АМБАРА В РЕДКОСТЬ</span><h2>Апгрейд растения</h2><p>Один экземпляр и монеты расходуются за попытку. При успехе получишь выбранное идеальное растение в амбар.</p>
        {!source?<div className="farm-game-empty"><Glyph name="box" size={40}/><p>Сначала собери урожай. Для попытки нужно растение из амбара.</p></div>:<>
          <div className="farm-upgrade-pair"><div><CropArt seed={source.seed} mutation={source.mutation}/><small>ОДИН ЭКЗЕМПЛЯР</small><strong>{name(source.seed)}</strong><span>{mutationNames[source.mutation]}</span></div><span className="farm-upgrade-arrow"><Glyph name="forward" size={30}/></span><div><CropArt seed={target} mutation={targetMutation}/><small>ИДЕАЛЬНОЕ РАСТЕНИЕ</small><strong>{name(target)}</strong><span>{mutationNames[targetMutation]}</span></div></div>
          <div className="farm-upgrade-selects"><label>Что отдаём<select value={source.id} onChange={e=>setSourceID(e.target.value)}>{state.barn.map(item=><option key={item.id} value={item.id}>{name(item.seed)} · {mutationNames[item.mutation]} · {gameNumber(Math.floor(item.value/item.quantity))} монет/шт.</option>)}</select></label><label>Что получим<select value={target} onChange={e=>setTarget(e.target.value)}>{crops.map(c=><option key={c.id} value={c.id} disabled={(seedLevels[c.id]??1)>level}>{c.name}{(seedLevels[c.id]??1)>level?` · с ${seedLevels[c.id]} уровня`:''}</option>)}</select></label><label>Мутация результата<select value={targetMutation} onChange={e=>setTargetMutation(e.target.value)}>{mutations.map(m=><option key={m.id} value={m.id} disabled={!m.unlocked}>{m.name} ×{m.multiplier}{!m.unlocked?` · генетика ${m.min_genetics}`:''}</option>)}</select></label></div>
          <div className="farm-upgrade-odds">{quote.isFetching?<span>Считаем шанс…</span>:quote.isError?<span role="alert">{quote.error.message}</span>:quote.data?<><strong>{(quote.data.chance_bp/100).toFixed(2)}%<small>ШАНС УСПЕХА</small></strong><div><span>Растение + монеты <b>{gameNumber(quote.data.source_value+quote.data.stake)}</b></span><span>Цена результата <b>{gameNumber(quote.data.target_value)}</b></span><small>Рыночные цены с учётом твоей лавки.</small></div></>:<span>Выбери ставку от {gameNumber(rules.min_stake)} монет.</span>}</div>
          <button className="farm-primary full" disabled={busy||!valid||!targetUnlocked||!quote.data||quote.isFetching||quote.isError} onClick={()=>onAction({kind:'plant_upgrade',item:source.id,seed:target,upgrade:targetMutation,stake:stake!})}>Попробовать апгрейд<Glyph name="spark" size={18}/></button>
        </>}
        {lastUpgrade&&<div className={`farm-upgrade-outcome ${lastUpgrade.status}`} key={lastUpgrade.id}><Glyph name={lastUpgrade.status==='won'?'check':'seed'} size={22}/><span><strong>{lastUpgrade.status==='won'?'Апгрейд удался!':'Последняя попытка не удалась'}</strong><small>{name(lastUpgrade.seed??'')} · {mutationNames[lastUpgrade.mutation??'none']} · шанс {((lastUpgrade.chance_bp??0)/100).toFixed(2)}%</small></span></div>}
      </div>}
    </section><aside className="farm-games-aside">
      <section className="farm-stake-panel"><span className="farm-eyebrow">ТВОЯ СТАВКА</span><label htmlFor="farm-game-stake">Сколько монет поставим?</label><div className="farm-stake-input"><Glyph name="coin" size={22}/><input id="farm-game-stake" type="number" min={rules.min_stake} max={rules.max_stake} step={1} value={stakeText} disabled={running&&mode==='crash'} onChange={e=>setStakeText(e.target.value)}/></div><div className="farm-stake-presets">{[100,1000,5000,10000].map(n=><button key={n} disabled={running&&mode==='crash'} onClick={()=>setStakeText(String(n))}>{gameNumber(n)}</button>)}</div><p>От {gameNumber(rules.min_stake)} до {gameNumber(rules.max_stake)} монет.{stake!==null&&state.coins<stake?' На балансе не хватает монет.':''}</p></section>
      <section className="farm-game-rules"><h3><Glyph name="book" size={18}/>Правила и шансы</h3>{mode==='crash'?<><p>Вероятность дойти до ×2: {percent(crashReturn/2)}%. До ×3: {percent(crashReturn/3)}%. До ×5: {percent(crashReturn/5)}%. До ×10: {percent(crashReturn/10)}%.</p><p>{percent(100-crashReturn)}% раундов заканчиваются сразу на ×1.00. Теоретический возврат до {crashReturn}% ставки. Выплата округляется вниз до целой монеты.</p></>:mode==='flip'?<><p>У каждой стороны шанс 50%. При ставке 1 000 с каждого победитель получает {gameNumber(coinflipPayout(1000,rules.flip_fee_percent))}.</p><p>Комиссия {rules.flip_fee_percent}% от общего банка. Сторону выбирает создатель, соперник получает противоположную. Выплата округляется вниз до целой монеты.</p></>:<><p>Шанс рассчитывается сервером из цены одного растения, ставки и рыночной цены цели.</p><p>Коэффициент {rules.upgrade_return_percent}%, максимум {(rules.upgrade_max_chance_bp/100).toFixed(0)}% успеха. При неудаче растение и ставка расходуются. Трофеи на витрине не участвуют.</p></>}<small>Выигрыши игр не прибавляются к выручке в топе фермеров.</small></section>
    </aside></div>
    <section className="farm-game-history"><header className="farm-section-heading spaced"><div><h2>Последние раунды</h2><p>Результаты сохраняются после закрытия страницы.</p></div></header>{!history.length?<div className="farm-game-empty">Здесь появится твой первый результат.</div>:<div className="farm-history-list">{[...history].reverse().slice(0,12).map(result=><article key={result.id} className={result.status}><Glyph name={result.kind==='crash'?'lamp':result.kind==='flip'?'sun':'spark'} size={19}/><span><strong>{result.kind==='crash'?'Переросток':result.kind==='flip'?'Коинфлип':'Апгрейд'}</strong><small>{new Date(result.at).toLocaleTimeString('ru-RU',{hour:'2-digit',minute:'2-digit'})} · ставка {gameNumber(result.stake)}</small></span><b>{result.kind==='upgrade'?result.status==='won'?'Растение':'Неудача':result.payout?`+${gameNumber(result.payout)}`:'0'}{result.multiplier_bp?<small>{multiplierText(result.multiplier_bp)}</small>:null}</b></article>)}</div>}</section>
    {reveal&&<GameReveal key={reveal.id} result={reveal} name={name(reveal.seed??'')} play={play} onClose={()=>setReveal(null)}/>}
  </>;
}
