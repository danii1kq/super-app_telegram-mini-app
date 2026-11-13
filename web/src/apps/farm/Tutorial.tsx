import { useState } from 'react';
import { Glyph } from './art';

/**
 * Tutorial is the first-run walkthrough. It replaces the old wall of text in
 * the help dialog: five short steps, one idea each, with the actual mechanics
 * stated plainly. A new player should be able to plant their first seed after
 * reading it, and should know why the barn and orders matter.
 */
type Step = {
  icon: string;
  eyebrow: string;
  title: string;
  body: string;
  points?: string[];
};

const steps: Step[] = [
  {
    icon: 'sprout',
    eyebrow: 'ШАГ 1 · ПОСАДКА',
    title: 'Посади первое семя',
    body: 'Нажми на пустую грядку и выбери семя. Растение созреет через три часа и никогда не испортится, даже если ты вернёшься завтра.',
    points: ['У тебя уже есть семена в запасе', 'Две первые грядки ждут с готовым урожаем'],
  },
  {
    icon: 'water',
    eyebrow: 'ШАГ 2 · УХОД',
    title: 'Выбери уход при посадке',
    body: 'Уход действует весь цикл, возвращаться для полива не нужно.',
    points: ['Полив бесплатный и повышает качество', 'Удобрение за 12 монет добавляет 25% урожайности', 'Лампа за 20 монет повышает шанс мутации'],
  },
  {
    icon: 'spark',
    eyebrow: 'ШАГ 3 · МУТАЦИИ',
    title: 'Иногда вырастает чудо',
    body: 'Каждое растение может оказаться мутантом. Мутация умножает цену, от ×2 у росистого до ×12 у радужного. Редкие виды открываются улучшением генетики.',
    points: ['Качество тоже влияет: отборное ×2, идеальное ×3', 'Каждая находка попадает в коллекцию навсегда'],
  },
  {
    icon: 'box',
    eyebrow: 'ШАГ 4 · АМБАР И ЗАКАЗЫ',
    title: 'Не продавай всё сразу',
    body: 'В амбаре есть кнопка «продать всё», но выгоднее сначала посмотреть на заказы деревни. Они платят больше рынка за конкретные растения.',
    points: ['Три заказа в день, каждый платит один раз', 'Заказ на качество принимает этот грейд или лучше'],
  },
  {
    icon: 'trophy',
    eyebrow: 'ШАГ 5 · ЧТО ДАЛЬШЕ',
    title: 'Копи на большое',
    body: 'Монеты уходят в улучшения, наборы и экспедиции. Впереди 50 уровней фермы: новые сорта и ступени прокачки открываются с опытом. Заходи за ежедневной наградой, собирай коллекцию из 216 находок и расти в недельном топе.',
    points: ['Бонусы роста фиксируются при посадке, лавка действует при продаже', 'Лаборатория скрещивает два одинаковых семени в гибрид'],
  },
];

export function Tutorial({onDone}:{onDone:()=>void}) {
  const [index, setIndex] = useState(0);
  const step = steps[index] ?? steps[0]!;
  const last = index === steps.length - 1;
  return <div className="farm-tutorial">
    <div className="farm-tutorial-head">
      <span className="farm-tutorial-icon"><Glyph name={step.icon} size={26}/></span>
      <span className="farm-eyebrow">{step.eyebrow}</span>
    </div>
    <h2>{step.title}</h2>
    <p>{step.body}</p>
    {step.points && <ul className="farm-tutorial-points">{step.points.map(point => <li key={point}><Glyph name="check" size={13}/>{point}</li>)}</ul>}
    <div className="farm-tutorial-dots" aria-label={`Шаг ${index + 1} из ${steps.length}`}>
      {steps.map((item, i) => <button key={item.eyebrow} className={i === index ? 'active' : ''} aria-label={`Шаг ${i + 1}`} onClick={() => setIndex(i)}/>)}
    </div>
    <div className="farm-tutorial-actions">
      <button className="farm-text-button" onClick={onDone}>Пропустить</button>
      {last
        ? <button className="farm-primary" onClick={onDone}>Начать играть<Glyph name="forward" size={15}/></button>
        : <button className="farm-primary" onClick={() => setIndex(index + 1)}>Дальше<Glyph name="forward" size={15}/></button>}
    </div>
  </div>;
}
