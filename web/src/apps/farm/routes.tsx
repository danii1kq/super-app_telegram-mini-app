import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate } from 'react-router-dom';
import { useSession } from '@/app/session';
import { BackButton } from '@/shared/telegram/hooks';
import { useFeedback } from '@/shared/ui/toast';
import { actOnFarm, countdown, getFarm, getLeaderboard, type FarmAction, type FarmResponse } from './api';
import { CropArt, Glyph, SeedPacket } from './art';
import { useFarmSound } from './useFarmSound';
import { PackOpening } from './PackOpening';
import { Themes } from './Themes';
import { growthLabel } from './GrowingPlant';
import { Laboratory } from './Laboratory';
import { Showcase } from './Showcase';
import { Atmosphere } from './Atmosphere';
import { Upgrades } from './Upgrades';
import { LevelPanel } from './LevelPanel';
import { OrderBoardPanel } from './OrderBoard';
import { DailyPanel } from './DailyPanel';
import { Tutorial } from './Tutorial';
import { Expeditions } from './Expeditions';
import { GardenScenery } from './GardenScenery';
import { Games } from './Games';
import { mutationNames, plantCharacter, plantWeight } from './character';
import './farm.css';
import './collection.css';
import './mutations.css';
import './worlds.css';
import './expeditions.css';
import './garden-visuals.css';
import './games.css';
import './reveals.css';
import './progression.css';

const tabs = [
  ['garden', 'garden', 'Моя ферма'], ['games','game','Игры'], ['seeds', 'seed', 'Семена'], ['collection', 'book', 'Коллекция'],
  ['lab', 'spark', 'Лаборатория'], ['explore', 'explore', 'Экспедиции'], ['showcase', 'trophy', 'Витрина'],
  ['upgrades', 'upgrade', 'Улучшения'], ['themes', 'spark', 'Темы'], ['top', 'trophy', 'Топ фермеров'],
] as const;
type Tab = typeof tabs[number][0];
type Modal = { kind: 'plant'; slot: number } | { kind: 'pack'; id: string } | { kind: 'drops'; drops: string[]; pack: string } | { kind: 'help' };
const qualityNames: Record<string, string> = { normal: 'Обычное', choice: 'Отборное', perfect: 'Идеальное' };
const rarityNames: Record<string, string> = { common: 'Обычное', uncommon: 'Необычное', rare: 'Редкое', legendary: 'Легендарное', hybrid: 'Гибрид' };
const careOptions = [
  { id: 'water', icon: 'water', name: 'Полив', cost: 0, description: '+5% идеального, +10% отборного' },
  { id: 'soil', icon: 'soil', name: 'Удобрение', cost: 12, description: '+25% урожайности' },
  { id: 'lamp', icon: 'lamp', name: 'Лампа', cost: 20, description: '+5% к шансу мутации' },
];
const number = (value: number) => value.toLocaleString('ru-RU');

function Coins({ value }: { value: number }) { return <span className="farm-coins"><Glyph name="coin" size={17}/>{number(value)}</span>; }

function Dialog({ title, children, onClose, immersive = false }: { title: string; children: ReactNode; onClose: () => void; immersive?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { const dialog = ref.current; dialog?.showModal(); return () => dialog?.close(); }, []);
  return <dialog ref={ref} className={`farm-dialog ${immersive ? 'farm-opening-dialog' : ''}`} aria-labelledby="farm-dialog-title" onCancel={(event) => { event.preventDefault(); onClose(); }} onClick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
    <div className="farm-dialog-inner"><header><h2 id="farm-dialog-title">{title}</h2><button className="farm-icon-button" onClick={onClose} aria-label="Закрыть"><Glyph name="close"/></button></header>{children}</div>
  </dialog>;
}

export default function FarmRoutes() {
  const session = useSession();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const cache = useQueryClient();
  const sound = useFarmSound();
  const [effect, setEffect] = useState<{ kind: string; slot?: number; coins: number } | null>(null);
  const [tab, setTab] = useState<Tab>(() => new URLSearchParams(window.location.search).get('tab') === 'games' ? 'games' : 'garden');
  const [modal, setModal] = useState<Modal | null>(null);
  const [seed, setSeed] = useState('carrot');
  const [care, setCare] = useState('water');
  const [filter, setFilter] = useState('none');
  const [seedFilter, setSeedFilter] = useState('all');
  const [tick, setTick] = useState(Date.now());
  const markSeen = () => { try { window.localStorage.setItem('srufarm-tutorial', '1'); } catch { /* private mode */ } };
  const queryKey = ['farm', session.me?.id];
  const boardKey = ['farm-leaderboard', session.me?.id];
  const farm = useQuery({ queryKey, queryFn: getFarm, enabled: session.status === 'ready', refetchInterval: (query) => query.state.data?.state.crash?.status === 'running' ? 350 : tab === 'games' ? 2000 : 30_000 });
  const leaderboard = useQuery({ queryKey: boardKey, queryFn: getLeaderboard, enabled: session.status === 'ready', refetchInterval: 60_000 });
  const mutation = useMutation({
    mutationFn: actOnFarm,
    onMutate: () => { void cache.cancelQueries({ queryKey }); },
    onSuccess: (data: FarmResponse, action: FarmAction) => {
      cache.setQueryData(queryKey, data);
      setEffect({ kind: action.kind, slot: action.slot, coins: data.state.coins - (farm.data?.state.coins ?? data.state.coins) });
      if (action.kind !== 'pack') {
        if (action.kind === 'harvest') {
          const collected = data.state.barn.slice(farm.data?.state.barn.length ?? 0);
          const special = collected.find(item => item.mutation === 'rainbow') ?? collected.find(item => item.mutation === 'ancient' || item.mutation === 'crystal') ?? collected.find(item => item.mutation !== 'none') ?? collected[0];
          sound.play(special ? plantCharacter(special.seed,special.mutation).sound : 'harvest');
        } else sound.play(action.kind === 'crash_cashout' || action.kind === 'plant_upgrade' || action.kind === 'flip_join' ? 'tap' : action.kind === 'crash_start' || action.kind === 'flip_create' || action.kind === 'flip_cancel' ? 'plant' : action.kind === 'theme' ? 'tap' : action.kind === 'redeem' || action.kind === 'breed' ? 'upgrade' : action.kind === 'keep' || action.kind === 'claim_breed' || action.kind === 'claim_level' || action.kind === 'fulfill_order' || action.kind === 'claim_daily' || action.kind === 'claim_expedition' ? 'rare' : action.kind === 'return_trophy' || action.kind === 'start_expedition' ? 'plant' : action.kind);
      }
      if (action.kind.startsWith('flip_')) void cache.invalidateQueries({ queryKey: ['farm-coinflips', session.me?.id] });
      if (action.kind === 'sell' || action.kind === 'fulfill_order') void cache.invalidateQueries({ queryKey: boardKey });
      if (action.kind === 'plant') setModal(null);
      if (action.kind === 'pack') setModal({ kind: 'drops', drops: data.drops ?? [], pack: action.pack ?? 'garden' });
      else if (action.kind !== 'plant_upgrade' && action.kind !== 'flip_join') feedback(data.message);
    },
    onError: (error: Error) => { feedback(error.message, 'error'); void cache.invalidateQueries({ queryKey }); void cache.invalidateQueries({ queryKey: ['farm-coinflips', session.me?.id] }); },
  });
  useEffect(() => { const timer = window.setInterval(() => setTick(Date.now()), 1000); return () => window.clearInterval(timer); }, []);
  useEffect(() => { if (!effect) return; const timer = window.setTimeout(() => setEffect(null), 1100); return () => window.clearTimeout(timer); }, [effect]);
  // First run only: a brand new player lands in the garden with no idea what to
  // do. Show the walkthrough once, then never again unless they open the rules.
  useEffect(() => {
    try { if (window.localStorage.getItem('srufarm-tutorial')) return; } catch { return; }
    setModal({ kind: 'help' });
  }, []);
  const act = (action: FarmAction) => { if (!mutation.isPending) { sound.prime(); mutation.mutate(['crash_start','crash_cashout','plant_upgrade','flip_create','flip_join','flip_cancel'].includes(action.kind) ? {...action,request_id:crypto.randomUUID()} : action); } };
  const back = () => navigate('/apps');

  if (session.status !== 'ready' || !farm.data) {
    const failed = farm.isError || session.status === 'error' || session.status === 'outside';
    return <div className="farm-app farm-loading"><BackButton onClick={back}/><Glyph name="sprout" size={48}/><h1>SruFarm</h1><p>{failed ? 'Не удалось открыть ферму' : 'Готовим твой маленький сад…'}</p>
      {failed && <><button className="farm-primary" onClick={() => { session.retry(); void farm.refetch(); }}>Попробовать ещё</button><Link to="/apps">Вернуться к приложениям</Link></>}
    </div>;
  }

  const { state, crops, recipes = [], mutations = [], packs, upgrades = [], themes = [], stars_available = false, server_time, level, daily, expeditions = [], games, seed_levels = {}, pack_levels = {}, bonuses } = farm.data;
  const allCrops = [...crops, ...recipes.map(recipe => recipe.crop)];
  const collectionSize = allCrops.length * Object.keys(mutationNames).length;
  const showcaseCapacity = 6 + (state.upgrades.gallery ?? 0) * 2;
  const now = Date.parse(server_time) + Math.max(0, tick - farm.dataUpdatedAt);
  const ready = state.plots.filter((plot) => plot && Date.parse(plot.ready_at) <= now).length;
  const occupied = state.plots.filter(Boolean).length;
  const rawBarnValue = state.barn.reduce((sum, item) => sum + item.value, 0);
  const marketBonus = bonuses.market_percent;
  const barnValue = Math.floor(rawBarnValue * (100 + marketBonus) / 100);
  const totalSeeds = Object.values(state.seeds).reduce((sum, count) => sum + count, 0);
  const myRank = leaderboard.data?.entries.find((entry) => entry.mine);
  const cropByID = (id: string) => allCrops.find((crop) => crop.id === id)!;
  const openPlant = (slot: number) => { setSeed(allCrops.find((crop) => (state.seeds[crop.id] ?? 0) > 0)?.id ?? 'carrot'); setCare('water'); setModal({ kind: 'plant', slot }); };
  const heading = {
    games: ['Рискни монетами.', 'Поймай свой момент.', 'Три игры: разгон растения, коинфлип с соперником и апгрейд урожая.'],
    garden: ['Маленькая ферма.', 'Большие находки.', 'Посади что-нибудь хорошее. Остальное сделает время.'],
    seeds: ['Всё начинается', 'с одного семечка.', 'Выбирай знакомые сорта или доверься удаче.'],
    collection: ['Твои маленькие', 'открытия.', 'Каждое новое растение остаётся здесь. Даже после продажи.'],
    upgrades: ['Немного заботы.', 'Больше чудес.', 'Улучшай ферму и повышай шансы на особенный урожай.'],
    explore: ['За калиткой', 'целый мир.', 'Отправляйся за новыми семенами. Три часа в пути, и новые находки уже у тебя.'],
    lab: ['У повторов', 'новое будущее.', 'Скрещивай одинаковые семена и открывай необычные гибриды.'],
    showcase: ['Сохрани то,', 'что удивило.', 'Твои любимые экземпляры. Маленький музей большой удачи.'],
    themes: ['Другая атмосфера.', 'Твоя любимая ферма.', 'Шесть миров, от тропического острова до полярного сада. Меняй всю атмосферу фермы.'],
    top: ['Растём вместе.', 'Соревнуемся честно.', 'Собирай урожай, продавай и поднимайся в недельном топе.'],
  }[tab];

  const seedGrid = (selectable: boolean) => <div className={`farm-seed-grid ${selectable ? 'selectable' : ''}`}>{(selectable ? allCrops.filter(crop => (state.seeds[crop.id] ?? 0) > 0) : crops.filter(crop => seedFilter === 'all' || crop.rarity === seedFilter)).map((crop) => <button key={crop.id} disabled={selectable ? !state.seeds[crop.id] : mutation.isPending || state.coins < crop.seed_price || level.level < (seed_levels[crop.id] ?? 1)} className={`farm-seed-card ${selectable && seed === crop.id ? 'selected' : ''}`} onClick={() => selectable ? setSeed(crop.id) : act({ kind: 'buy', seed: crop.id })}>
    <span className={`farm-rarity-dot ${crop.rarity}`}/><span className="farm-owned">{state.seeds[crop.id] ?? 0} шт.</span><CropArt seed={crop.id}/><strong>{crop.name}</strong><small>{selectable ? `Продажа от ${crop.price}` : rarityNames[crop.rarity]}</small>
    {!selectable && <span className="farm-seed-buy"><span>{level.level < (seed_levels[crop.id] ?? 1) ? `С ${seed_levels[crop.id]} уровня` : 'Купить семя'}</span><Coins value={crop.seed_price}/></span>}
  </button>)}</div>;

  const activeTheme = state.theme ?? 'garden';
  const weather = ({garden:'Хороший день',tropical:'Морской бриз',lunar:'Лунная ночь',enchanted:'Светлячки',autumn:'Золотой час',frost:'Тихий снег'} as Record<string,string>)[activeTheme] ?? 'Хороший день';
  const emblem = ({garden:'carrot',tropical:'dragonfruit',lunar:'moonflower',enchanted:'crystal_bloom',autumn:'pumpkin',frost:'blueberry'} as Record<string,string>)[activeTheme] ?? 'carrot';
  return <div className={`farm-app theme-${state.theme ?? 'garden'} ${effect ? `farm-action-${effect.kind}` : ''}`}>
    <BackButton onClick={back}/>
    <Atmosphere/>
    <header className="farm-topbar"><div className="farm-topbar-inner">
      <Link to="/apps" className="farm-back" aria-label="К приложениям"><Glyph name="arrow" size={19}/></Link>
      <Link className="farm-brand" to="/apps/farm" onClick={() => setTab('garden')}><span className="farm-brand-mark"><Glyph name="sprout" size={25}/></span>Sru<span>Farm</span><span className="farm-beta">BETA</span></Link>
      <div className="farm-topbar-right"><button className="farm-sound-toggle" onClick={sound.toggle} aria-label={sound.enabled ? 'Выключить звуки' : 'Включить звуки'} aria-pressed={sound.enabled} title={sound.enabled ? 'Звуки включены' : 'Звуки выключены'}><Glyph name={sound.enabled ? 'sound' : 'muted'} size={18}/></button><span className="farm-wallet"><Coins value={state.coins}/>{effect?.kind === 'sell' && <span className="farm-coin-pop" aria-hidden="true">+{number(effect.coins)}</span>}</span><button className="farm-avatar" onClick={() => setTab('top')} aria-label="Мой рейтинг">{(session.me?.public_name ?? 'Ф').slice(0,1).toUpperCase()}</button></div>
    </div></header>

    <div className="farm-layout">
      <nav className="farm-tabs" aria-label="Разделы фермы">{tabs.map(([id, icon, label]) => <button key={id} className={tab === id ? 'active' : ''} aria-current={tab === id ? 'page' : undefined} onClick={() => setTab(id)}><Glyph name={icon}/><span>{label}</span>{id === 'collection' && <small>{Object.keys(state.collection).length}</small>}</button>)}</nav>
      <main>
        <section className="farm-intro"><div><span className="farm-eyebrow"><span/> ТВОЙ УГОЛОК СПОКОЙСТВИЯ</span><h1>{heading[0]}<br/><em>{heading[1]}</em></h1><p>{heading[2]}</p></div><div className="farm-intro-art"><span className="farm-intro-orbit"/><CropArt seed={tab === 'seeds' ? 'dragonfruit' : tab === 'collection' ? 'blueberry' : emblem}/><span className="farm-art-spark"><Glyph name="spark" size={24}/></span></div></section>

        {tab === 'garden' && <div className="farm-columns"><div>
          <section className="farm-garden-panel">
            <header className="farm-section-heading"><div><h2>Мой участок <span className="farm-count">{occupied}/{state.plots.length}</span></h2><p>{ready ? `${ready} грядки готовы к сбору` : occupied ? 'Тихо растёт что-то прекрасное' : 'Свободные грядки ждут семена'}</p></div><span className="farm-weather"><Glyph name={activeTheme === 'lunar' ? 'spark' : 'sun'}/>{weather}</span></header>
            <div className="farm-garden">
              <GardenScenery/>
              <div className="farm-plots">{state.plots.map((plot, slot) => {
                const mature = plot && Date.parse(plot.ready_at) <= now;
                const remaining = plot ? Date.parse(plot.ready_at) - now : 0;
                const duration = plot ? Math.max(1, Date.parse(plot.ready_at) - Date.parse(plot.planted_at)) : 1;
                const progress = plot ? Math.min(100, Math.max(0, (1 - remaining / duration) * 100)) : 0;
                return <button key={slot} className={`farm-plot ${plot ? mature ? 'ready' : 'growing' : 'empty'} ${effect?.kind === 'plant' && effect.slot === slot ? 'fresh' : ''}`} disabled={mutation.isPending} onClick={() => plot ? mature ? act({ kind: 'harvest' }) : feedback(`Растёт ${cropByID(plot.seed).name.toLowerCase()}. ${growthLabel(progress / 100)}. До сбора ${countdown(remaining)}`) : openPlant(slot)} aria-label={plot ? `${cropByID(plot.seed).name}, ${mature ? 'собрать урожай' : `${growthLabel(progress / 100)}, до сбора ${countdown(remaining)}`}` : `Посадить на грядке ${slot + 1}`}>
                  <span className="farm-plot-soil"/>
                  {plot ? <><CropArt seed={plot.seed} growing={!mature} progress={progress / 100} variant={slot}/><span className="farm-plot-name">{cropByID(plot.seed).name}</span><span className={`farm-plot-status ${mature ? 'mature' : ''}`}><Glyph name={mature ? 'check' : 'clock'} size={13}/>{mature ? 'Собрать' : countdown(remaining)}</span>{!mature && <span className="farm-growth-track"><span style={{ width: `${progress}%` }}/></span>}<span className="farm-plot-care"><Glyph name={plot.care} size={15}/></span></> : <><span className="farm-plot-plus"><Glyph name="plus" size={22}/></span><span className="farm-plot-name">Посадить семя</span><span className="farm-empty-caption">Свободная грядка</span></>}
                </button>;
              })}</div>
            </div>
            <footer className="farm-garden-footer"><span><Glyph name="clock" size={17}/>Один урожай каждые 3 часа</span><button className="farm-primary" disabled={!ready || mutation.isPending} onClick={() => act({ kind: 'harvest' })}><Glyph name="seed" size={17}/>{ready ? `Собрать всё · ${ready}` : 'Урожай растёт'}</button></footer>
          </section>
          <div className="farm-gentle-note"><Glyph name="sprout" size={18}/><p>{state.xp === 0 ? 'Два первых растения уже выросли. Это подарок для знакомства с фермой.' : 'Растения растут, пока ты занят. Готовый урожай не портится и всегда дождётся тебя.'}</p><button onClick={() => setModal({ kind: 'help' })}>Как играть<Glyph name="arrow" size={13}/></button></div>
          <section className="farm-inventory"><header className="farm-section-heading"><div><h2>Запас семян <span className="farm-count">{totalSeeds}</span></h2><p>Что посадим в следующий раз?</p></div><button className="farm-text-button" onClick={() => setTab('seeds')}>В магазин <span>↗</span></button></header><div className="farm-seed-strip">{allCrops.filter((crop) => state.seeds[crop.id]).map((crop) => <button key={crop.id} onClick={() => { const empty = state.plots.findIndex((plot) => !plot); if (empty >= 0) { openPlant(empty); setSeed(crop.id); } else feedback('Все грядки заняты. Семена сохранятся до следующей посадки.'); }}><CropArt seed={crop.id}/><span>{crop.name}<small>{state.seeds[crop.id]} шт.</small></span></button>)}{totalSeeds === 0 && <p>Семена закончились. Новые ждут в магазине.</p>}</div></section>
        </div><aside className="farm-sidebar">
          <section className="farm-barn"><header className="farm-section-heading"><div><h2><Glyph name="box"/>Амбар</h2><p>Собрано с заботой</p></div><span className="farm-count">{state.barn.reduce((sum,item) => sum + item.quantity,0)}</span></header>
            {state.barn.length ? <div className="farm-barn-items">{state.barn.map((item) => <div className="farm-barn-item" key={item.id}><CropArt seed={item.seed} mutation={item.mutation}/><div><strong>{cropByID(item.seed).name}<small> ×{item.quantity}</small></strong><span>{item.mutation !== 'none' ? mutationNames[item.mutation] : qualityNames[item.quality]}{item.weight ? ` · ${plantWeight(item.weight)}` : ''}</span><small className="farm-barn-character">{plantCharacter(item.seed,item.mutation).label}</small></div><div className="farm-barn-actions"><Coins value={item.value}/><button title="Сохранить один экземпляр на витрине" aria-label={`На витрину: ${cropByID(item.seed).name}`} disabled={mutation.isPending || !item.id || (state.showcase?.length ?? 0) >= showcaseCapacity} onClick={() => act({kind:'keep',item:item.id})}><Glyph name="trophy" size={14}/>На витрину</button></div></div>)}</div> : <div className="farm-barn-empty"><Glyph name="box" size={38}/><strong>Здесь будет твой урожай</strong><p>Собери созревшие растения,<br/>чтобы продать их за монеты.</p></div>}
            <div className="farm-barn-total"><span>Стоимость урожая{marketBonus > 0 ? ` · лавка +${marketBonus}%` : ''}</span><Coins value={barnValue}/></div><button className="farm-primary full" disabled={!barnValue || mutation.isPending} onClick={() => act({ kind: 'sell' })}>Продать урожай<Glyph name="forward" size={16}/></button>
          </section>
          <button className="farm-pack-teaser" onClick={() => setModal({ kind: 'pack', id: 'garden' })}><div><span className="farm-eyebrow">НЕМНОГО МАГИИ</span><h3>А что внутри?</h3><p>Три семени.<br/>Шанс на редкую находку.</p><span className="farm-teaser-cta">Открыть набор <Glyph name="forward" size={14}/></span></div><SeedPacket/></button>
          <OrderBoardPanel board={state.orders} barn={state.barn} crops={allCrops} busy={mutation.isPending} onAction={(order) => act({ kind: 'fulfill_order', item: order })}/>
          <DailyPanel daily={daily} crops={allCrops} busy={mutation.isPending} onClaim={() => act({ kind: 'claim_daily' })}/>
          <LevelPanel level={level} busy={mutation.isPending} onClaim={(value) => act({ kind: 'claim_level', level: value })}/>
        </aside></div>}

        {tab === 'seeds' && <>
          <div className="farm-content-banner"><Glyph name="explore" size={25}/><div><strong>За новыми сортами можно отправиться в путь</strong><p>Лаванда, манго, лотос и карамбола ждут в экспедициях. Награды известны заранее.</p></div><button className="farm-secondary" onClick={() => setTab('explore')}>Выбрать маршрут<Glyph name="forward" size={14}/></button></div>
          <section className="farm-packs-grid">{packs.map((pack) => <article key={pack.id} className={`farm-pack-card ${pack.id}`}><div className="farm-pack-visual"><SeedPacket variant={pack.id}/><span className="farm-pack-orbit"/></div><div className="farm-pack-copy"><span className="farm-eyebrow">3 СЕМЕНИ ВНУТРИ</span><h2>{pack.name}</h2><span className="farm-pack-chance">{pack.weights.reduce((sum,weight,i) => sum + (crops[i]?.rarity === 'legendary' ? weight : 0),0)}% легендарного семени</span><p>{pack.id === 'orchard' ? 'Шесть новых сортов: от сладкого перца до карамболы. 25% легендарного семени до прокачки селекции.' : pack.id === 'lunar' ? 'Только необычные и выше. Легендарное не позже шестого семени.' : pack.id === 'garden' ? 'Знакомые любимцы и маленький шанс на большое чудо.' : 'Больше необычных сортов. Для тех, кто любит открывать новое.'}</p><button className="farm-primary" onClick={() => setModal({ kind: 'pack', id: pack.id })}>{level.level < (pack_levels[pack.id] ?? 1) ? `С ${pack_levels[pack.id]} уровня` : 'Посмотреть набор'}<Coins value={pack.price}/></button></div></article>)}</section>
          <header className="farm-section-heading spaced"><div><h2>Знаешь, что хочешь вырастить?</h2><p>Исходные сорта продаются здесь. Гибриды создаются в лаборатории.</p></div></header><div className="farm-filters" aria-label="Редкость семян">{([['all','Все сорта'],['common','Обычные'],['uncommon','Необычные'],['rare','Редкие'],['legendary','Легендарные']] as const).map(([id,label])=><button key={id} aria-pressed={seedFilter===id} className={seedFilter===id?'active':''} onClick={()=>setSeedFilter(id)}>{label}</button>)}</div>{seedGrid(false)}
        </>}

        {tab === 'collection' && <><div className="farm-collection-summary"><Glyph name="book" size={28}/><div><strong>{Object.keys(state.collection).length}<span> / {collectionSize} открытий</span></strong><p>{allCrops.length} видов. Восемь обликов. Целая коллекция историй.</p></div><div className="farm-collection-progress"><span style={{ width: `${Object.keys(state.collection).length / (collectionSize) * 100}%` }}/></div></div><div className="farm-filters" aria-label="Мутация">{Object.entries(mutationNames).map(([id,name]) => <button key={id} aria-pressed={filter === id} className={filter === id ? 'active' : ''} onClick={() => setFilter(id)}>{name}</button>)}</div><div className="farm-collection-grid">{allCrops.map((crop) => { const discovered = state.collection[`${crop.id}:${filter}`]; return <article key={crop.id} className={`farm-discovery ${discovered ? '' : 'locked'}`}><span className={`farm-rarity-label ${crop.rarity}`}>{rarityNames[crop.rarity]}</span><CropArt seed={crop.id} mutation={filter}/><h3>{crop.name}</h3><p>{discovered ? `${mutationNames[filter]} · ${qualityNames[discovered.quality]}` : 'Ещё не найдено'}</p><span className="farm-discovered-mark">{discovered ? <><Glyph name="check" size={14}/>В коллекции</> : <><Glyph name="seed" size={14}/>Вырасти, чтобы открыть</>}</span></article>; })}</div></>}

        {tab === 'upgrades' && <Upgrades upgrades={upgrades} level={level} crops={crops} packs={packs} seedLevels={seed_levels} packLevels={pack_levels} bonuses={bonuses} mutations={mutations} coins={state.coins} busy={mutation.isPending} onUpgrade={id => act({kind:'upgrade',upgrade:id})}/> }

        {tab === 'games' && games && <Games state={state} crops={allCrops} mutations={mutations} rules={games} serverTime={server_time} receivedAt={farm.dataUpdatedAt} me={session.me!.id} level={level.level} seedLevels={seed_levels} busy={mutation.isPending} onAction={act} play={sound.play}/>}
        {tab === 'explore' && <Expeditions routes={expeditions} journey={state.expedition} crops={allCrops} coins={state.coins} level={level.level} now={now} busy={mutation.isPending} onAction={act}/>}
        {tab === 'lab' && <Laboratory state={state} recipes={recipes} crops={crops} now={now} busy={mutation.isPending} onAction={act}/> }
        {tab === 'showcase' && <Showcase capacity={showcaseCapacity} trophies={state.showcase ?? []} crops={allCrops} busy={mutation.isPending} onAction={act} onGarden={() => setTab('garden')}/> }

        {tab === 'themes' && <Themes themes={themes} state={state} busy={mutation.isPending} starsAvailable={stars_available} onAction={act} onPaid={() => { void farm.refetch(); }}/> }

        {tab === 'top' && <div className="farm-ranking-layout"><section className="farm-ranking"><header className="farm-section-heading"><div><span className="farm-eyebrow">НЕДЕЛЬНАЯ ЯРМАРКА</span><h2>Топ фермеров</h2><p>{leaderboard.data ? `До нового сезона: ${Math.max(0, Math.ceil((Date.parse(leaderboard.data.ends_at) - now) / 86400_000))} дн. · сброс в понедельник, 00:00 UTC` : 'Загружаем результаты…'}</p></div><Glyph name="trophy" size={32}/></header><div className="farm-ranking-labels"><span>МЕСТО / ФЕРМЕР</span><span>ПРОДАНО НА</span></div>{leaderboard.isError ? <div className="farm-ranking-empty"><p>Не удалось загрузить рейтинг</p><button className="farm-secondary" onClick={() => void leaderboard.refetch()}>Повторить</button></div> : leaderboard.data?.entries.length ? leaderboard.data.entries.map((entry) => <div key={entry.id} className={`farm-ranking-row ${entry.mine ? 'mine' : ''}`}><span className={`farm-rank rank-${entry.rank}`}>{entry.rank}</span><span className="farm-ranking-avatar">{entry.name.slice(0,1).toUpperCase()}</span><strong>{entry.name}{entry.mine && <small>Это ты</small>}</strong><Coins value={entry.revenue}/></div>) : <div className="farm-ranking-empty"><Glyph name="trophy" size={44}/><h3>Первый урожай. Первое место.</h3><p>Продай урожай, чтобы открыть рейтинг этой недели.</p><button className="farm-primary" onClick={() => setTab('garden')}>На ферму</button></div>}</section><aside className="farm-ranking-about"><span className="farm-ranking-big"><Glyph name="trophy" size={56}/></span><h2>{myRank ? `Ты на ${myRank.rank} месте` : 'Твоё место ждёт'}</h2><p>В зачёт идёт стоимость проданных растений за эту неделю. Покупка семян и наборов очков не добавляет.</p><div><span>Продано за всё время</span><Coins value={state.sold}/></div><small>В топе только настоящие игроки. При равном результате выше тот, кто раньше зарегистрировался.</small></aside></div>}
      </main>
      <footer className="farm-page-footer"><span><Glyph name="sprout" size={15}/> SruFarm · Растём в своём ритме</span><button onClick={() => setModal({ kind: 'help' })}>Правила игры <Glyph name="plus" size={14}/></button></footer>
    </div>

    {modal && <Dialog immersive={modal.kind === 'drops'} title={modal.kind === 'plant' ? `Посадка · грядка ${modal.slot + 1}` : modal.kind === 'drops' ? 'Открытие набора' : modal.kind === 'pack' ? packs.find((pack) => pack.id === modal.id)!.name : 'Маленький путеводитель'} onClose={() => { if (modal.kind === 'help') markSeen(); setModal(null); }}>
      {mutation.isError && <p className="farm-inline-error" role="alert">{mutation.error.message}</p>}
      {modal.kind === 'plant' && <><p className="farm-dialog-subtitle">Выбери семя. Через 3 часа здесь вырастет что-то хорошее.</p>{seedGrid(true)}{totalSeeds === 0 ? <button className="farm-primary full" onClick={() => { setModal(null); setTab('seeds'); }}>За семенами в магазин</button> : <><h3 className="farm-care-title">Немного заботы</h3><div className="farm-care-options">{careOptions.map((option) => <button key={option.id} aria-pressed={care === option.id} className={care === option.id ? 'selected' : ''} onClick={() => setCare(option.id)}><Glyph name={option.icon} size={24}/><strong>{option.name}</strong><small>{option.cost ? `${option.cost} монет` : 'Бесплатно'}</small><span>{option.description}</span></button>)}</div><button className="farm-primary full large" disabled={mutation.isPending || !state.seeds[seed] || state.coins < (careOptions.find((option) => option.id === care)?.cost ?? 0)} onClick={() => act({ kind: 'plant', slot: modal.slot, seed, care })}><Glyph name="sprout"/>{mutation.isPending ? 'Сажаем…' : 'Посадить семя'}<span>03:00</span></button><p className="farm-dialog-footnote">Уход действует весь цикл. Возвращаться для полива не нужно.</p></>}</>}
      {modal.kind === 'pack' && (() => { const pack = packs.find((item) => item.id === modal.id)!; return <><div className="farm-pack-modal-art"><SeedPacket variant={pack.id}/></div><p className="farm-dialog-subtitle center">Три случайных семени. У каждого есть шанс стать началом чего-то особенного.</p><div className="farm-pack-odds">{crops.flatMap((crop,i) => !pack.weights[i] ? [] : [<span key={crop.id}><span className={`farm-rarity-dot ${crop.rarity}`}/>{crop.name}<strong>{pack.weights[i]}%</strong></span>])}</div><div className="farm-pity"><Glyph name="spark" size={18}/><p>До {pack.guarantee === 'legendary' ? 'легендарного' : 'редкого'} семени: <strong>{pack.pity_after - (state.pity[pack.id] ?? 0)} семян</strong><small>Каждое {pack.pity_after}-е семя без {pack.guarantee === 'legendary' ? 'легендарной' : 'редкой'} находки гарантированно будет {pack.guarantee === 'legendary' ? 'легендарным' : 'редким или лучше'}. Счётчик отдельный для каждого набора.</small></p></div><button className="farm-primary full large" disabled={mutation.isPending || state.coins < pack.price || level.level < (pack_levels[pack.id] ?? 1)} onClick={() => act({ kind: 'pack', pack: pack.id })}>{level.level < (pack_levels[pack.id] ?? 1) ? `С ${pack_levels[pack.id]} уровня` : mutation.isPending ? 'Открываем…' : state.coins < pack.price ? 'Не хватает монет' : 'Открыть набор'}<Coins value={pack.price}/></button></>; })()}
      {modal.kind === 'drops' && <PackOpening drops={modal.drops.map(cropByID)} pack={packs.find(pack => pack.id === modal.pack)!} play={sound.play} prime={sound.prime} stop={sound.stop} soundEnabled={sound.enabled} onToggleSound={sound.toggle} onClose={() => { setModal(null); setTab('garden'); }} onAgain={() => setModal({ kind: 'pack', id: modal.pack })}/>}
      {modal.kind === 'help' && <Tutorial onDone={() => { setModal(null); markSeen(); }}/>}
    </Dialog>}
  </div>;
}
