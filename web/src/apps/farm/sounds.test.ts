import assert from 'node:assert/strict';
import { test } from 'node:test';
import { playFarmSound, type FarmSound } from './sounds.ts';

test('sounds are brief, quiet, and release their audio nodes', () => {
  for (const sound of ['plant', 'harvest', 'sell', 'buy', 'pack', 'rare', 'upgrade', 'tap'] satisfies FarmSound[]) {
    const notes: { start: number; stop: number; peak: number; disconnected: boolean }[] = [];
    const fake = {
      state: 'running', currentTime: 10, destination: {},
      createOscillator() {
        const note = { start: 0, stop: 0, peak: 0, disconnected: false }; notes.push(note);
        return { type: '', frequency: { setValueAtTime() {}, exponentialRampToValueAtTime() {} },
          connect() {}, disconnect() { note.disconnected = true; }, onended: null as (() => void) | null,
          start(at: number) { note.start = at; }, stop(at: number) { note.stop = at; this.onended?.(); } };
      },
      createGain() { return { gain: { setValueAtTime() {}, linearRampToValueAtTime(peak: number) { notes[notes.length - 1]!.peak = peak; }, exponentialRampToValueAtTime() {} }, connect() {}, disconnect() {} }; },
    };
    playFarmSound(fake as unknown as AudioContext, sound);
    assert.ok(notes.length > 0);
    for (const note of notes) {
      assert.ok(note.stop - 10 < 1, 'effect must finish in under a second');
      assert.ok(note.peak > 0 && note.peak <= .04, 'effect must remain quiet');
      assert.ok(note.disconnected, 'finished oscillator must disconnect');
    }
    fake.state = 'suspended';
    const count = notes.length;
    playFarmSound(fake as unknown as AudioContext, sound);
    assert.equal(notes.length, count, 'never queue notes in a suspended context');
  }
});
