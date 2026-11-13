import assert from 'node:assert/strict';
import {test} from 'node:test';
import {canCraft,cooldown,levelProgress,questionSeconds} from './display.ts';
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
