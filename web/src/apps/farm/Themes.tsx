import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { isMiniApp, tg } from '@/shared/telegram/sdk';
import { useFeedback } from '@/shared/ui/toast';
import { CropArt, Glyph } from './art';
import { getThemeInvoice, type FarmAction, type FarmState, type FarmTheme } from './api';
import './themes.css';

export function Themes({ themes,state,busy,starsAvailable,onAction,onPaid }: {
  themes: FarmTheme[]; state: FarmState; busy: boolean; starsAvailable: boolean;
  onAction: (action: FarmAction) => void; onPaid: () => void;
}) {
  const [code,setCode] = useState('');
  const feedback = useFeedback();
  const native = isMiniApp() && Boolean(tg?.openInvoice);
  const invoice = useMutation({mutationFn:getThemeInvoice,onSuccess:data=>{
    tg?.openInvoice?.(data.url,status=>{
      if(status==='paid' || status==='pending') {onPaid();feedback('Проверяем подтверждение оплаты. Тема появится после подтверждения Telegram.');}
      else if(status==='failed')feedback('Не удалось оплатить тему','error');
    });
  },onError:(error:Error)=>feedback(error.message,'error')});
  return <>
    <div className="farm-themes-grid">{themes.map(theme=>{
      const owned=state.owned_themes?.includes(theme.id) ?? theme.id==='garden';
      const active=(state.theme ?? 'garden')===theme.id;
      return <article key={theme.id} className={`farm-theme-card ${theme.id} ${active?'active':''}`}>
        <div className={`farm-theme-preview ${theme.id}`} aria-hidden="true"><span className="farm-theme-sky"/><span className="farm-theme-tree"/><span className="farm-theme-island"/><div className="farm-theme-mini-plots"><CropArt seed={({garden:'carrot',tropical:'dragonfruit',lunar:'moonflower',enchanted:'crystal_bloom',autumn:'pumpkin',frost:'blueberry'} as Record<string,string>)[theme.id] ?? 'carrot'}/><CropArt seed={theme.id==='tropical'?'dragonfruit':'strawberry'}/></div><span className="farm-theme-preview-label">{({garden:'ORIGINAL GARDEN',tropical:'TROPICAL ISLAND',lunar:'NIGHT GARDEN',enchanted:'SPIRIT FOREST',autumn:'GOLDEN HOUR',frost:'POLAR GARDEN'} as Record<string,string>)[theme.id]}</span></div>
        <div className="farm-theme-info"><span className="farm-eyebrow">{active?'ТВОЯ ТЕКУЩАЯ ТЕМА':owned?'ОТКРЫТА НАВСЕГДА':'ОСОБОЕ ОФОРМЛЕНИЕ'}</span><h2>{theme.name}</h2><p>{theme.description}</p>
          {owned?<button className="farm-primary full" disabled={active || busy} onClick={()=>onAction({kind:'theme',theme:theme.id})}>{active?'Применена':'Применить тему'}<Glyph name={active?'check':'spark'} size={17}/></button>:<><button className="farm-primary full" disabled={!starsAvailable || !native || invoice.isPending} onClick={()=>invoice.mutate(theme.id)}>{invoice.isPending?'Готовим счёт…':`Открыть за ${theme.stars} Stars`}<Glyph name="spark" size={17}/></button><small>{!starsAvailable?'Оплата Stars пока не подключена. Доступен промокод.':!native?'Для оплаты открой ферму в Telegram.':'Тема откроется после подтверждённой оплаты.'}</small></>}
        </div>
      </article>;
    })}</div>
    <form className="farm-promo" onSubmit={event=>{event.preventDefault();if(code.trim())onAction({kind:'redeem',code:code.trim()});}}><div><span className="farm-promo-icon"><Glyph name="spark" size={26}/></span><h3>Есть секретный код?</h3><p>Активируй промокод, чтобы открыть новую тему.</p></div><div className="farm-promo-field"><label className="sr-only" htmlFor="farm-promo-code">Промокод</label><input id="farm-promo-code" maxLength={64} value={code} onChange={event=>setCode(event.target.value)} placeholder="Введи промокод" autoComplete="off" autoCapitalize="characters" spellCheck={false}/><button className="farm-primary" disabled={busy || !code.trim()} type="submit">Активировать<Glyph name="forward" size={14}/></button></div></form>
    <p className="farm-theme-note"><Glyph name="sprout" size={16}/>Темы меняют оформление фермы. Условия выращивания и шансы остаются общими для всех.</p>
  </>;
}
