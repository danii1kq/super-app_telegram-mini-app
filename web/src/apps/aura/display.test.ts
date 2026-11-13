import assert from 'node:assert/strict';
import {test} from 'node:test';
import {canCraft,cooldown,levelProgress,questionSeconds,relicComparison,lootRevealDelay} from './display.ts';
test('question timer expires at the server deadline and never becomes negative',()=>{
  const now=Date.parse('2026-11-13T10:00:00Z');
  assert.equal(questionSeconds('2026-11-13T10:00:00.001Z',now),1);
  assert.equal(questionSeconds('2026-11-13T10:00:00Z',now),0);
  assert.equal(questionSeconds('invalid',now),0);
  assert.equal(cooldown('2026-11-13T10:00:59Z',now),'59 с');
});
test('crafting requires all materials as well as the full aura price',()=>{
  assert.equal(canCraft(100,{crypt:3},100,{crypt:3}),true);
  assert.equal(canCraft(99,{crypt:3},100,{crypt:3}),false);
  assert.equal(canCraft(100,{crypt:3},100,{crypt:3,void:1}),false);
});
test('XP bar uses progress inside the earned level',()=>{
  assert.equal(levelProgress(80,2,120),0);
  assert.equal(levelProgress(100,2,120),20);
  assert.equal(levelProgress(1000,2,120),120);
});
test('item comparison never ranks unrelated effects or the same equipped item',()=>{
  const item={key:'candidate',passive:'damage',value:.25};
  assert.equal(relicComparison(item,{key:'old',passive:'damage',value:.18}),7);
  assert.equal(relicComparison(item,{key:'old',passive:'damage',value:.3}),-5);
  assert.equal(relicComparison(item,{key:'old',passive:'crit',value:.1}),null);
  assert.equal(relicComparison(item,{...item,value:.1}),null);
  assert.equal(relicComparison(item),null);
});
test('rarity reveal remains short while giving higher rarity longer suspense',()=>{
  const durations=['E','D','C','B','A','S','SS','SSS'].map(lootRevealDelay);
  assert.ok(durations.every((duration,index)=>duration>0&&duration<=3000&&(index===0||duration>durations[index-1]!)));
  assert.equal(lootRevealDelay('legacy-unknown'),lootRevealDelay('E'));
});
