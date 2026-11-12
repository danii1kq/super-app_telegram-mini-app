import type { Mutation } from './api';
import { CropArt, Glyph } from './art';

export function MutationGuide({mutations}:{mutations:Mutation[]}) {
  return <section className="farm-mutation-guide">
    <header className="farm-section-heading spaced"><div><h2>Каким будет следующий урожай?</h2><p>Сначала выпадает шанс мутации, затем её вид. Генетика открывает новые варианты.</p></div></header>
    <div className="farm-mutation-grid">{mutations.map(kind=><article key={kind.id} className={`farm-mutation-card ${kind.unlocked?'':'locked'}`}>
      <CropArt seed="tomato" mutation={kind.id}/><div><h3>{kind.name}</h3><strong>×{kind.multiplier} к цене</strong><small>{kind.unlocked?<><Glyph name="check" size={12}/>Может вырасти</>:<>Генетика, уровень {kind.min_genetics}</>}</small></div>
    </article>)}</div>
  </section>;
}
