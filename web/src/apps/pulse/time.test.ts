import { test } from 'node:test';
import assert from 'node:assert/strict';
import { localDateTime, meetingInput, minutesLeft } from './time.ts';

test('Pulse remains active for the whole final minute and expires at its exact end', () => {
  const end = '2026-11-12T12:00:00Z';
  assert.equal(minutesLeft(end, Date.parse(end) - 59_000), 1);
  assert.equal(minutesLeft(end, Date.parse(end) - 1), 1);
  assert.equal(minutesLeft(end, Date.parse(end)), 0);
  assert.equal(minutesLeft(end, Date.parse(end) + 1000), 0);
});

test('calendar confirmation validates the interval and preserves local wall-clock input', () => {
  const instant = new Date('2026-11-12T12:34:00Z');
  assert.equal(new Date(localDateTime(instant)).getTime(), instant.getTime());
  const input = meetingInput(' Coffee ', '2026-11-12T12:00', '2026-11-12T13:00');
  assert.equal(input?.title, 'Coffee');
  assert.equal(Date.parse(input!.ends_at) - Date.parse(input!.starts_at), 3_600_000);
  assert.equal(meetingInput(' ', '2026-11-12T12:00', '2026-11-12T13:00'), null);
  assert.equal(meetingInput('Coffee', 'invalid', '2026-11-12T13:00'), null);
  assert.equal(meetingInput('Coffee', '2026-11-12T13:00', '2026-11-12T12:00'), null);
  assert.equal(meetingInput('Coffee', '2026-11-12T12:00', '2027-01-01T12:00'), null);
});
