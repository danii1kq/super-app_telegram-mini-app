import assert from 'node:assert/strict';
import { test } from 'node:test';
import { gameMultiplier, gameStake, multiplierText } from './gameDisplay.ts';

test('display grows from the server start time, stays capped, and handles clock skew',()=>{
  const start='2026-11-13T00:00:00Z',now=Date.parse(start);
  assert.equal(gameMultiplier(now-5000,start,.08,1000),100);
  assert.equal(gameMultiplier(now+10000,start,.08,1000),222);
  assert.equal(gameMultiplier(now+100000,start,.08,1000),1000);
  assert.equal(gameMultiplier(now,'invalid',.08,1000),100);
  assert.equal(multiplierText(222),'×2.22');
});

test('stake input accepts only whole amounts within the rules',()=>{
  for(const input of ['', ' ', '-100','99','100001','100.5','NaN','Infinity'])assert.equal(gameStake(input,100,100000),null);
  assert.equal(gameStake('100',100,100000),100);
  assert.equal(gameStake('100000',100,100000),100000);
});
