import { test } from 'node:test';
import assert from 'node:assert/strict';
import { receiptPreview, parseReceiptMinor } from './receiptPreview.ts';

test('receipt prices accept printed free modifiers and exact minor units', () => {
  assert.equal(parseReceiptMinor('0,00'), 0);
  assert.equal(parseReceiptMinor('0'), 0);
  assert.equal(parseReceiptMinor('2 349,96'), 234996);
  assert.equal(parseReceiptMinor('1950.02'), 195002);
  for (const invalid of ['', '-1', '1,234', 'unknown', '10000000000.01']) assert.equal(parseReceiptMinor(invalid), null);
  const result = receiptPreview([{ key: 'doner', amount: 234996, units: { '1': 1 } }, { key: 'free', amount: parseReceiptMinor('0,00')!, units: { '1': 1 } }], 0, 0, 234996, '1');
  assert.equal(result.valid, true);
  assert.equal(result.difference, 0);
  assert.deepEqual(result.unallocated, []);
});

test('receipt preview preserves personal items and exact discount rounding', () => {
  const result = receiptPreview([{ key: 'pizza', amount: 400000, units: { '1': 1, '2': 1 } }, { key: 'burger', amount: 200000, units: { '2': 1 } }], 0, 0, 600000, '1');
  assert.deepEqual(result.perPerson, { '1': 200000, '2': 400000 });
  assert.deepEqual(result.owes, { '2': 400000 });
  assert.equal(result.difference, 0);
  assert.equal(result.valid, true);
  const discount = receiptPreview([{ key: 'x', amount: 200, units: { '10': 1, '2': 1 } }], 1, 0, 199, '2');
  assert.deepEqual(discount.perPerson, { '2': 99, '10': 100 });
});

test('receipt preview uses bigint products and rejects invalid draft limits', () => {
  const result = receiptPreview([{ key: 'x', amount: 999_999_999_999, units: { '1': 999, '2': 998 } }], 999_999_999_998, 0, 1, '1');
  assert.equal(result.sum, 1);
  assert.equal(result.difference, 0);
  assert.equal(result.valid, true);
  assert.equal(receiptPreview([{ key: 'x', amount: Number.MAX_SAFE_INTEGER + 1, units: { '1': 1 } }], 0, 0, 1, '1').valid, false);
  assert.equal(receiptPreview([{ key: 'x', amount: 100, units: { '1': 1000 } }], 0, 0, 100, '1').valid, false);
  assert.equal(receiptPreview([{ key: 'x', amount: 100, units: { '1': 1 } }], 101, 0, 0, '1').valid, false);
  assert.deepEqual(receiptPreview([{ key: 'missing', amount: 100, units: {} }], 0, 0, 100, '1').unallocated, ['missing']);
});
