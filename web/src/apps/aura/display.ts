export function questionSeconds(deadline:string,now:number):number {
  const end=Date.parse(deadline);
  return Number.isFinite(end)?Math.max(0,Math.ceil((end-now)/1000)):0;
}
export function cooldown(readyAt:string|null|undefined,now:number):string {
  const seconds=readyAt?questionSeconds(readyAt,now):0;
  if(!seconds)return '';
  if(seconds<60)return `${seconds} с`;
  const minutes=Math.ceil(seconds/60);
  return minutes>=60?`${Math.floor(minutes/60)} ч ${minutes%60} мин`:`${minutes} мин`;
}
export function canCraft(aura:number,materials:Record<string,number>,price:number,cost:Record<string,number>):boolean {
  return aura>=price&&Object.entries(cost).every(([key,count])=>(materials[key]??0)>=count);
}
export function levelProgress(total:number,level:number,need:number):number {
  let spent=0;
  for(let current=1;current<level;current++)spent+=80+(current-1)*40;
  return Math.max(0,Math.min(need,total-spent));
}

export function relicComparison(candidate:{key:string;passive:string;value:number},equipped?:{key:string;passive:string;value:number}):number|null {
  if(!equipped||candidate.key===equipped.key||candidate.passive!==equipped.passive)return null;
  return Math.round((candidate.value-equipped.value)*10000)/100;
}

export function lootRevealDelay(rank:string):number {
  return ({E:850,D:1000,C:1150,B:1350,A:1600,S:2000,SS:2300,SSS:2600} as Record<string,number>)[rank]??850;
}

export const materialLabel = (key:string):string => ({bone:'Костяные осколки',ember:'Угли кузницы',void:'Эссенция пустоты',seal:'Печати аномалий'} as Record<string,string>)[key]??key;
