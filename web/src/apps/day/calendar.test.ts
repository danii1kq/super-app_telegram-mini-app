import { test } from 'node:test';
import assert from 'node:assert/strict';
import { addDays, eventsOnDay, localInput, parseLocalInput, sortEvents, startOfWeek } from './calendar.ts';

test('calendar uses local week boundaries across DST and exclusive event endings', () => {
  const previousZone = process.env.TZ;
  process.env.TZ = 'America/New_York';
  try {
    const monday = startOfWeek(new Date('2026-03-08T12:00:00-04:00'));
    assert.equal(localInput(monday), '2026-03-02T00:00');
    assert.equal((addDays(monday, 7).getTime() - monday.getTime()) / 3_600_000, 167);
    const spanning = { id: '1', starts_at: '2026-03-07T23:00:00-05:00', ends_at: '2026-03-09T00:00:00-04:00', all_day: false };
    const earlier = { id: '2', starts_at: '2026-03-07T09:00:00-05:00', ends_at: '2026-03-07T10:00:00-05:00', all_day: false };
    assert.deepEqual(eventsOnDay([spanning, earlier], new Date('2026-03-07T00:00:00-05:00')).map((event) => event.id), ['2', '1']);
    assert.deepEqual(eventsOnDay([spanning], new Date('2026-03-08T00:00:00-05:00')), [spanning]);
    assert.deepEqual(eventsOnDay([spanning], new Date('2026-03-09T00:00:00-04:00')), []);
    assert.equal(parseLocalInput('2026-03-08T02:30'), undefined);
    assert.equal(parseLocalInput('2026-02-30T10:00'), undefined);
    assert.equal(parseLocalInput('not a date'), undefined);
    assert.equal(parseLocalInput('2026-03-08T10:30')?.toISOString(), '2026-03-08T14:30:00.000Z');
    assert.deepEqual(sortEvents([spanning, earlier]), [earlier, spanning]);
    assert.deepEqual([spanning, earlier].map((event) => event.id), ['1', '2']);
  } finally {
    if (previousZone === undefined) delete process.env.TZ; else process.env.TZ = previousZone;
  }
});
