import type { FarmSound } from './sounds';

export const mutationNames: Record<string,string> = {
  none:'Обычное', giant:'Гигантское', dew:'Росистое', gold:'Золотое',
  electric:'Грозовое', crystal:'Кристальное', ancient:'Древнее', rainbow:'Радужное',
};

export const hybridArt: Record<string,{ base: string; hue: number }> = {
  aurora_root:{base:'carrot',hue:285}, jade_tomato:{base:'tomato',hue:75},
  moonberry:{base:'strawberry',hue:250}, ghost_pumpkin:{base:'pumpkin',hue:35},
  solar_corn:{base:'corn',hue:335}, crystal_bloom:{base:'moonflower',hue:35},
  flame_pepper:{base:'pepper',hue:335}, honey_pear:{base:'pear',hue:340},
  astral_lotus:{base:'lotus',hue:70},
};

export function plantCharacter(seed: string,mutation = 'none'): { label: string; sound: FarmSound; kind: string } {
  if(mutation==='rainbow')return{label:'Переливается всеми цветами радуги',sound:'legendary',kind:'rainbow'};
  if(mutation==='ancient')return{label:'Хранит тёплый свет древних рун',sound:'rare',kind:'ancient'};
  if(mutation==='electric')return{label:'Потрескивает маленькими молниями',sound:'hit5',kind:'electric'};
  if(mutation==='dew')return{label:'Прячет утреннюю росу на листьях',sound:'plant',kind:'dew'};
  if(mutation==='crystal')return{label:'Звенит, как маленький колокольчик',sound:'rare',kind:'crystal'};
  if(mutation==='gold')return{label:'Ловит каждый солнечный луч',sound:'sell',kind:'gold'};
  if(mutation==='giant')return{label:'Еле помещается на грядке',sound:'hit1',kind:'giant'};
  if(seed==='moonflower' || seed==='moonberry' || seed==='crystal_bloom' || seed==='lotus' || seed==='astral_lotus')return{label:'Тихо светится изнутри',sound:'tap',kind:'moon'};
  if(seed==='dragonfruit' || seed==='solar_corn' || seed==='flame_pepper' || seed==='starfruit')return{label:'Согревает своими искорками',sound:'harvest',kind:'ember'};
  if(seed==='lavender')return{label:'Покачивается, будто слышит музыку',sound:'rare',kind:'leaf'};
  if(seed==='pear' || seed==='honey_pear' || seed==='mango')return{label:'Любит греться на солнышке',sound:'harvest',kind:'round'};
  if(seed==='pumpkin' || seed==='ghost_pumpkin')return{label:'Любит покачиваться на боку',sound:'harvest',kind:'round'};
  return{label:'Тянется навстречу твоей заботе',sound:'harvest',kind:'leaf'};
}

export function plantWeight(grams: number): string {
  if(!grams)return'Размер не измерен';
  return grams>=1000 ? `${(grams/1000).toLocaleString('ru-RU',{maximumFractionDigits:2})} кг` : `${grams} г`;
}
