export function gameMultiplier(now:number,startedAt:string,growth:number,cap:number):number {
  const elapsed=Math.max(0,(now-Date.parse(startedAt))/1000);
  if(!Number.isFinite(elapsed))return 100;
  return Math.min(cap,Math.floor(Math.exp(Math.min(elapsed*growth,Math.log(cap/100)))*100));
}

export function gameStake(value:string,min:number,max:number):number|null {
  if(!value.trim())return null;
  const stake=Number(value);
  return Number.isInteger(stake)&&stake>=min&&stake<=max?stake:null;
}

export const gameNumber=(value:number)=>value.toLocaleString('ru-RU');
export const multiplierText=(bp:number)=>`×${(bp/100).toFixed(2)}`;
