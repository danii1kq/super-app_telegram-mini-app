import assert from 'node:assert/strict';
import { test } from 'node:test';
import type { Crop, Pack, Upgrade } from './api';
import { upcomingUnlocks } from './progression.ts';

const crops:Crop[] = [
  {id:'carrot',name:'Морковь',rarity:'common',price:32,seed_price:12},
  {id:'blueberry',name:'Голубика',rarity:'rare',price:210,seed_price:110},
  {id:'moonflower',name:'Лунный цветок',rarity:'legendary',price:800,seed_price:450},
];
const packs:Pack[] = [{id:'lunar',name:'Лунный набор',price:900,weights:[],pity_after:6,guarantee:'legendary'}];
const upgrades:Upgrade[] = [
  {id:'water',name:'Полив',icon:'water',description:'',benefit:'',level:5,max_level:20,cost:5400,required_level:12},
  {id:'land',name:'Грядки',icon:'garden',description:'',benefit:'',level:4,max_level:4,cost:7500,required_level:14},
];

test('roadmap groups openings by level and excludes owned upgrade caps and passed gates',()=>{
  assert.deepEqual(upcomingUnlocks(3,crops,packs,upgrades,{carrot:1,blueberry:6,moonflower:12},{lunar:12}),[
    {level:6,names:['Голубика']},
    {level:12,names:['Лунный цветок','Лунный набор','Полив 6']},
  ]);
  assert.deepEqual(upcomingUnlocks(12,crops,packs,upgrades,{carrot:1,blueberry:6,moonflower:12},{lunar:12}),[]);
});

test('roadmap only shows the next three future level milestones in ascending order',()=>{
  const extra:Crop[] = [4,7,9,13].map(level=>({id:`crop${level}`,name:`Сорт ${level}`,rarity:'rare',price:100,seed_price:50}));
  const levels = Object.fromEntries([4,7,9,13].map(level=>[`crop${level}`,level]));
  assert.deepEqual(upcomingUnlocks(3,extra.reverse(),[],[],levels,{}).map(unlock=>unlock.level),[4,7,9]);
});
