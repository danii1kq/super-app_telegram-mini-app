import { useEffect, useId, useState, type CSSProperties } from 'react';
import { CropArt, Glyph } from './art';
import type { Crop, Pack } from './api';
import type { FarmSound } from './sounds';
import { revealPlan } from './opening';

const names: Record<string, string> = { common: 'ОБЫЧНОЕ', uncommon: 'НЕОБЫЧНОЕ', rare: 'РЕДКОЕ', legendary: 'ЛЕГЕНДАРНОЕ' };

function SeedCore({ cracks }: { cracks: number }) {
  const id = useId();
  return <svg className="farm-seed-core" viewBox="0 0 240 240" fill="none" aria-hidden="true">
    <defs><linearGradient id={`${id}-shell`} x1="50" y1="30" x2="180" y2="210" gradientUnits="userSpaceOnUse"><stop stopColor="currentColor" stopOpacity=".75"/><stop offset=".4" stopColor="currentColor" stopOpacity=".17"/><stop offset="1" stopColor="currentColor" stopOpacity=".55"/></linearGradient></defs>
    <path d="M120 26C93 53 47 91 47 141c0 41 28 72 73 72s73-31 73-72c0-48-41-88-73-115Z" fill={`url(#${id}-shell)`} stroke="currentColor" strokeWidth="2"/>
    <path d="M120 27c-24 49-29 127 0 185m0-185c40 63 41 127 0 185M56 113l64 29 65-28M55 170l65-28 65 29" stroke="currentColor" strokeWidth="1" opacity=".45"/>
    <path d="M96 61C77 85 64 105 64 127" stroke="white" strokeOpacity=".5" strokeWidth="4" strokeLinecap="round"/>
    <path d="m120 85 10 28 29 9-29 9-10 29-10-29-29-9 29-9 10-28Z" fill="currentColor" opacity=".8"/>
    {cracks >= 2 && <path d="m120 30-8 36 15 22-17 31 14 25-8 39 4 28" stroke="#fff8d9" strokeWidth={cracks >= 4 ? 3.5 : 1.7} strokeLinejoin="round"/>}
    {cracks >= 3 && <path d="m125 87 30-6 18 14m-63 24-30 7-27 32m71-14 27 13 27-1" stroke="#fff8d9" strokeWidth="2" strokeLinejoin="round"/>}
  </svg>;
}

export function PackOpening({ drops, pack, play, prime, stop, soundEnabled, onToggleSound, onClose, onAgain }: {
  drops: Crop[]; pack: Pack; play: (sound: FarmSound) => void; prime: () => void; stop: () => void;
  soundEnabled: boolean; onToggleSound: () => void; onClose: () => void; onAgain: () => void;
}) {
  const [stage, setStage] = useState<'sealed' | 'opening' | 'revealed' | 'summary'>('sealed');
  const [index, setIndex] = useState(0);
  const [beat, setBeat] = useState(0);
  const [suspense, setSuspense] = useState(false);
  const current = drops[index];
  const plan = revealPlan(current?.rarity ?? 'common');
  const tier = stage === 'revealed' ? plan.tier : stage === 'opening' ? Math.floor(Math.max(0, beat - 1) * plan.tier / (plan.beats.length - 1)) : 0;

  useEffect(() => {
    if (!current) return;
    const sequence = revealPlan(current.rarity);
    if (stage === 'revealed') { play(sequence.sound); return stop; }
    if (stage !== 'opening') return;
    const timers = sequence.beats.map((at,i) => window.setTimeout(() => {
      setBeat(i + 1); play(`hit${i + 1}` as FarmSound);
    }, at));
    timers.push(window.setTimeout(() => { stop(); setSuspense(true); }, sequence.suspenseAt));
    timers.push(window.setTimeout(() => { setSuspense(false); setStage('revealed'); }, sequence.revealAt));
    return () => { timers.forEach(window.clearTimeout); stop(); };
  }, [stage, current?.id, current?.rarity, play, stop]);

  const open = () => { prime(); setBeat(0); setSuspense(false); setStage('opening'); };
  const next = () => {
    if (index + 1 >= drops.length) { setStage('summary'); return; }
    setIndex(index + 1); setBeat(0); setSuspense(false); setStage('sealed');
  };

  if (!current) return <button className="farm-primary full" onClick={onClose}>Вернуться на ферму</button>;

  return <section className={`farm-opening farm-summon ${stage} tier-${tier} ${suspense ? 'suspense' : ''} rarity-${current.rarity}`} style={{'--charge':beat} as CSSProperties} aria-label="Открытие семян">
    <div className="farm-opening-top"><span>{pack.name}</span><div><button aria-label={soundEnabled ? 'Выключить звук открытия' : 'Включить звук открытия'} onClick={onToggleSound}><Glyph name={soundEnabled ? 'sound' : 'muted'} size={15}/></button>{stage !== 'summary' && <button onClick={() => setStage('summary')}>Пропустить<Glyph name="arrow" size={12}/></button>}</div></div>
    {stage === 'summary' ? <div className="farm-opening-summary"><span className="farm-opening-kicker">ТВОЯ НОВАЯ ИСТОРИЯ</span><h3>Забирай находки.</h3><p>Все три семени уже в запасе.</p><div className="farm-opening-results">{drops.map((crop,i) => <article key={i} className={`farm-opening-result rarity-${crop.rarity}`} style={{ animationDelay: `${i * 140}ms` }}><span>{names[crop.rarity]}</span><CropArt seed={crop.id}/><strong>{crop.name}</strong><small>1 семя</small></article>)}</div><button className="farm-opening-primary" onClick={onClose}>На ферму<Glyph name="sprout"/></button><button className="farm-opening-again" onClick={onAgain}>Ещё один набор<Glyph name="box" size={16}/></button></div> : <>
      <div className="farm-opening-progress" aria-label={`Семя ${index + 1} из ${drops.length}`}>{drops.map((_,i) => <span key={i} className={i < index ? 'complete' : i === index ? 'current' : ''}>{i < index ? <Glyph name="check" size={13}/> : i + 1}</span>)}</div>
      <div className="farm-opening-stage">
        <svg className="farm-seed-sigil" viewBox="0 0 300 300" aria-hidden="true"><path d="m150 15 116 68v134L150 285 34 217V83Z"/><circle cx="150" cy="150" r="112" strokeDasharray="3 17"/><path d="m150 36 9 24-9 10-9-10Zm114 114-24 9-10-9 10-9ZM150 264l-9-24 9-10 9 10ZM36 150l24-9 10 9-10 9Z"/></svg>
        <div className="farm-seed-beam" aria-hidden="true"/>
        <div className="farm-reveal-splinters" key={`${index}-${stage}`} aria-hidden="true">{Array.from({length:8},(_,i)=><i key={i} style={{'--shard':i} as CSSProperties}/>)}</div>
        <div className="farm-summon-stars" aria-hidden="true">{Array.from({length:18},(_,i)=><i key={i} style={{left:`${(i*37+11)%100}%`,top:`${(i*29+7)%100}%`,animationDelay:`${i*.19}s`}}/>)}</div>
        <div className="farm-opening-glow" aria-hidden="true"/><div className="farm-opening-rays" aria-hidden="true"/>
        <div className="farm-opening-ring ring-a" aria-hidden="true"/><div className="farm-opening-ring ring-b" aria-hidden="true"/>
        {stage === 'opening' && beat > 0 && <div key={`${index}-${beat}`} className="farm-summon-pulse" aria-hidden="true"><span/><span/><i/></div>}
        <div className="farm-opening-particles" aria-hidden="true">{Array.from({ length:24 },(_,i)=><i key={i} style={{'--particle-angle':`${i*15}deg`,'--particle-delay':`${i%4*40}ms`} as CSSProperties}/>)}</div>
        {stage === 'revealed' ? <><div className="farm-summon-impact" aria-hidden="true"/><span className="farm-summon-backword" aria-hidden="true">{names[current.rarity]}</span><div className="farm-opening-revealed" key={index}><span className="farm-opening-rarity">{names[current.rarity]}</span><CropArt seed={current.id}/><h3>{current.name}</h3><span className="farm-opening-quantity">+1 семя в твой запас</span></div></> : <>
          <button className="farm-summon-core-button" onClick={open} disabled={stage === 'opening'} aria-label={`Раскрыть семя ${index + 1}`}><div className={`farm-summon-core ${stage === 'opening' ? 'energized' : ''}`} key={`${index}-${beat}`}><SeedCore cracks={beat}/></div></button>
          {stage === 'sealed' && <span className="farm-summon-touch"><Glyph name="spark" size={16}/>Коснись семени</span>}
        </>}
      </div>
      <div className="farm-opening-bottom" aria-live="polite">{stage === 'sealed' ? <><span className="farm-summon-caption">НАХОДКА {index + 1} / {drops.length}</span><h3>Что скрывается внутри?</h3><button className="farm-opening-primary" onClick={open}>Пробудить семя<Glyph name="spark"/></button></> : stage === 'opening' ? <><span className="farm-summon-caption">{suspense ? 'СЕЙЧАС…' : 'ЭНЕРГИЯ НАРАСТАЕТ'}</span><h3>{suspense ? 'Одно мгновение.' : ['Слышишь?','Что-то пробуждается.','Свет становится сильнее.','Это что-то особенное.','Невероятная сила.'][Math.max(0,beat-1)]}</h3><div className="farm-summon-energy" aria-hidden="true">{[1,2,3,4,5].map(n=><i key={n} className={n<=beat ? 'lit' : ''}/>)}</div></> : <><p>{current.rarity === 'legendary' ? 'Легендарная находка. Этот момент стоил ожидания.' : current.rarity === 'rare' ? 'Редкое семя. Твоей ферме повезло.' : 'Ещё одно начало для твоего сада.'}</p><button className="farm-opening-primary" onClick={next}>{index + 1 === drops.length ? 'Все находки' : 'Следующее семя'}<Glyph name={index + 1 === drops.length ? 'check' : 'seed'}/></button></>}</div>
    </>}
    <small className="farm-opening-saved"><Glyph name="check" size={12}/>Находки сохранены, даже если закрыть окно.</small>
  </section>;
}
