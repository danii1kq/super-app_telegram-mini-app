import type { Crop, Harvest, Order, OrderBoard } from './api';
import { CropArt, Glyph } from './art';

const qualityNames: Record<string, string> = { normal: 'Обычное', choice: 'Отборное', perfect: 'Идеальное' };
const mutationNames: Record<string, string> = {
  none: 'Обычное', giant: 'Гигантское', dew: 'Росистое', gold: 'Золотое',
  electric: 'Грозовое', crystal: 'Кристальное', ancient: 'Древнее', rainbow: 'Радужное',
};

/** Counts matching plants in the barn, mirroring the server's matching rules. */
function available(barn: Harvest[], order: Order): number {
  const rank: Record<string, number> = { normal: 0, choice: 1, perfect: 2 };
  return barn.reduce((total, item) => {
    if (item.seed !== order.seed) return total;
    if (order.quality && (rank[item.quality] ?? 0) < (rank[order.quality] ?? 0)) return total;
    if (order.mutation && item.mutation !== order.mutation) return total;
    return total + item.quantity;
  }, 0);
}

/**
 * OrderBoardPanel is the village board. It exists so the barn is not a single
 * "sell everything" button: some batches are worth holding for a specific order.
 */
export function OrderBoardPanel({board, barn, crops, busy, onAction}:{board:OrderBoard; barn:Harvest[]; crops:Crop[]; busy:boolean; onAction:(order:string)=>void}) {
  const done = new Set(board.fulfilled ?? []);
  return <section className="farm-orders">
    <header className="farm-section-heading"><div><h2><Glyph name="box" size={16}/>Заказы деревни</h2><p>Особые просьбы на сегодня. Платят больше рынка.</p></div><span className="farm-count">{done.size}/{board.orders.length}</span></header>
    <div className="farm-order-list">{board.orders.map(order => {
      const crop = crops.find(item => item.id === order.seed);
      const have = available(barn, order);
      const fulfilled = done.has(order.id);
      const ready = !fulfilled && have >= order.quantity;
      return <article key={order.id} className={`farm-order ${fulfilled ? 'done' : ready ? 'ready' : ''}`}>
        <CropArt seed={order.seed} mutation={order.mutation || 'none'}/>
        <div className="farm-order-copy">
          <strong>{crop?.name ?? order.seed}<small> ×{order.quantity}</small></strong>
          <span>{order.mutation ? mutationNames[order.mutation] : order.quality ? qualityNames[order.quality] : 'Любое качество'}</span>
          <small className="farm-order-have">{fulfilled ? 'Выполнен' : `В амбаре: ${have} из ${order.quantity}`}</small>
        </div>
        <div className="farm-order-side">
          <span className="farm-order-pay"><Glyph name="coin" size={13}/>{order.coins}</span>
          <button className="farm-secondary" disabled={busy || !ready} onClick={() => onAction(order.id)}>{fulfilled ? 'Готово' : ready ? 'Отдать' : 'Не хватает'}</button>
        </div>
      </article>;
    })}</div>
  </section>;
}
