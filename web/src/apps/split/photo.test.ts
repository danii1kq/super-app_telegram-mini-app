import { test } from 'node:test';
import assert from 'node:assert/strict';
import { cropPixels, rotatedSize } from './photo.ts';

test('crop stays inside the oriented image, including narrow and edge crops', () => {
  assert.deepEqual(cropPixels(1920, 2560, { x: 25, y: 10, width: 50, height: 80 }), { x: 480, y: 256, width: 960, height: 2048 });
  assert.deepEqual(cropPixels(100, 100, { x: 99, y: 99, width: 30, height: 30 }), { x: 99, y: 99, width: 1, height: 1 });
  assert.throws(() => cropPixels(0, 100, { x: 0, y: 0, width: 100, height: 100 }));
  assert.deepEqual(rotatedSize(8000, 4000, 1), { width: 2048, height: 4096 });
  assert.deepEqual(rotatedSize(1920, 2560, 4), { width: 1920, height: 2560 });
});
