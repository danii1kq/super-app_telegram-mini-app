import assert from 'node:assert/strict';
import { test } from 'node:test';
import { hybridArt, plantCharacter, plantWeight } from './character.ts';

test('mutations override the plant personality and select their own sound',()=>{
  assert.equal(plantCharacter('moonberry','crystal').kind,'crystal');
  assert.equal(plantCharacter('pumpkin','gold').sound,'sell');
  assert.equal(plantCharacter('carrot','giant').kind,'giant');
  assert.equal(plantCharacter('moonflower').kind,'moon');
  assert.equal(plantCharacter('dragonfruit').kind,'ember');
});

test('hybrids have drawable source species and trophy weights handle old saves',()=>{
  assert.equal(Object.keys(hybridArt).length,6);
  const sources=new Set(['carrot','tomato','strawberry','pumpkin','corn','moonflower']);
  for(const art of Object.values(hybridArt)){assert.ok(sources.has(art.base));assert.ok(Number.isFinite(art.hue));}
  assert.equal(plantWeight(0),'Размер не измерен');
  assert.equal(plantWeight(350),'350 г');
  assert.equal(plantWeight(7000),'7 кг');
});
