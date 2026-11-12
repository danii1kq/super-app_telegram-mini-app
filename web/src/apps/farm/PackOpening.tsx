import { useEffect, useState, type CSSProperties } from 'react';
import { CropArt, Glyph, SeedPacket } from './art';
import type { Crop } from './api';
import type { FarmSound } from './sounds';

const names: Record<string, string> = { common: 'ОБЫЧНОЕ', uncommon: 'НЕОБЫЧНОЕ', rare: 'РЕДКОЕ', legendary: 'ЛЕГЕНДАРНОЕ' };

export function PackOpening({ drops, exotic, play, onClose, onAgain }: {
  drops: Crop[]; exotic: boolean; play: (sound: FarmSound) => void; onClose: () => void; onAgain: () => void;
}) {
  const [stage, setStage] = useState<'sealed' | 'charging' | 'revealed' | 'summary'>('sealed');
  const [index, setIndex] = useState(0);
  const current = drops[index];

  useEffect(() => {
    if (stage !== 'charging' || !current) return;
    const timer = window.setTimeout(() => {
      setStage('revealed');
      play(current.rarity === 'legendary' ? 'legendary' : current.rarity === 'rare' ? 'rare' : 'harvest');
    }, current.rarity === 'legendary' ? 1050 : 650);
    return () => window.clearTimeout(timer);
  }, [stage, current, play]);

  const open = () => { play('pack'); setStage('charging'); };
  const next = () => {
    if (index + 1 >= drops.length) { setStage('summary'); return; }
    setIndex(index + 1); setStage('sealed');
  };

  if (!current) return <button className="farm-primary full" onClick={onClose}>Вернуться на ферму</button>;

  return <section className={`farm-opening ${stage} rarity-${current.rarity}`} aria-label="Открытие семян">
    <div className="farm-opening-top"><span>SRU SEEDS / {exotic ? 'EXOTIC' : 'GARDEN'}</span>{stage !== 'summary' && <button onClick={() => setStage('summary')}>Показать всё<Glyph name="arrow" size={12}/></button>}</div>
    {stage === 'summary' ? <div className="farm-opening-summary"><span className="farm-opening-kicker">НОВЫЕ ВОЗМОЖНОСТИ</span><h3>Хороший улов.</h3><p>Все семена уже в твоём запасе.</p><div className="farm-opening-results">{drops.map((crop,i) => <article key={i} className={`farm-opening-result rarity-${crop.rarity}`} style={{ animationDelay: `${i * 110}ms` }}><span>{names[crop.rarity]}</span><CropArt seed={crop.id}/><strong>{crop.name}</strong><small>1 семя</small></article>)}</div><button className="farm-opening-primary" onClick={onClose}>На ферму<Glyph name="sprout"/></button><button className="farm-opening-again" onClick={onAgain}>Посмотреть ещё набор<Glyph name="box" size={16}/></button></div> : <>
      <div className="farm-opening-progress" aria-label={`Семя ${index + 1} из ${drops.length}`}>{drops.map((_,i) => <span key={i} className={i < index ? 'complete' : i === index ? 'current' : ''}><Glyph name={i < index ? 'check' : 'seed'} size={14}/></span>)}</div>
      <div className="farm-opening-stage" key={`${index}-${stage}`}>
        <div className="farm-opening-glow" aria-hidden="true"/>
        <div className="farm-opening-rays" aria-hidden="true"/>
        <div className="farm-opening-ring ring-a" aria-hidden="true"/><div className="farm-opening-ring ring-b" aria-hidden="true"/>
        <div className="farm-opening-particles" aria-hidden="true">{Array.from({ length: 12 }, (_,i) => <i key={i} style={{ '--particle-angle': `${i * 30}deg`, '--particle-delay': `${i * 35}ms` } as CSSProperties}/>)}</div>
        {stage === 'sealed' ? <button className="farm-opening-sealed" onClick={open} aria-label={`Раскрыть семя ${index + 1}`}><SeedPacket exotic={exotic}/><span className="farm-opening-tap"><Glyph name="spark" size={18}/>Нажми на пакет</span></button> : stage === 'charging' ? <div className="farm-opening-charging"><Glyph name="seed" size={80}/></div> : <div className="farm-opening-revealed"><span className="farm-opening-rarity">{names[current.rarity]}</span><CropArt seed={current.id}/><h3>{current.name}</h3><span className="farm-opening-quantity">+1 семя в твой запас</span></div>}
      </div>
      <div className="farm-opening-bottom" aria-live="polite">{stage === 'sealed' ? <><h3>{index === 0 ? 'Пусть случится чудо.' : 'Что принесёт следующее?'}</h3><p>Внутри может быть твоя новая редкость.</p><button className="farm-opening-primary" onClick={open}>Раскрыть семя<Glyph name="spark"/></button></> : stage === 'charging' ? <><h3>Семя пробуждается…</h3><p>Ещё мгновение.</p></> : <><p>{current.rarity === 'legendary' ? 'Вот это находка. Твоя ферма будет сиять.' : current.rarity === 'rare' ? 'Особенное семя для особенной грядки.' : 'Новая жизнь начинается с маленького семечка.'}</p><button className="farm-opening-primary" onClick={next}>{index + 1 === drops.length ? 'Забрать всё' : 'Следующее семя'}<Glyph name={index + 1 === drops.length ? 'check' : 'seed'}/></button></>}</div>
    </>}
    <small className="farm-opening-saved"><Glyph name="check" size={12}/>Семена сохранены. Окно можно закрыть.</small>
  </section>;
}
