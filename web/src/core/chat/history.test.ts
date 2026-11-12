import { test } from 'node:test';
import assert from 'node:assert/strict';
import { orderedMessages } from './history.ts';

test('history joins older pages in sequence order and retains the newest duplicate', () => {
  const result = orderedMessages([
    { messages: [{ id: 'b', seq: '9007199254740993', text: 'updated' }, { id: 'c', seq: '9007199254740994', text: 'new' }] },
    { messages: [{ id: 'a', seq: '9007199254740992', text: 'old' }, { id: 'b', seq: '9007199254740993', text: 'stale' }] },
  ]);
  assert.deepEqual(result.map((message) => message.id), ['a', 'b', 'c']);
  assert.equal(result[1]?.text, 'updated');
  assert.deepEqual(orderedMessages([]), []);
});
