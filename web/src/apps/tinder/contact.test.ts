import assert from 'node:assert/strict';
import { test } from 'node:test';
import { telegramContact } from './contact.ts';

test('contact uses a public Telegram username, never an ID or arbitrary URL', () => {
  assert.equal(telegramContact('test_user'), 'https://t.me/test_user');
  for (const invalid of ['', '12345', 'https://evil.test', 'test/route', 'user?start=secret', '@name', 'abc']) assert.equal(telegramContact(invalid), '');
});
