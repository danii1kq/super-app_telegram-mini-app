import assert from 'node:assert/strict';
import { test } from 'node:test';
import { playFarmSound, soundNotes, type FarmSound } from './sounds.ts';
import { gameRevealPlan, revealPlan } from './opening.ts';

const sounds: FarmSound[] = ['plant','harvest','sell','buy','pack','rare','legendary','upgrade','tap','hit1','hit2','hit3','hit4','hit5'];

function fakeAudio() {
  const notes: { start: number; stop: number; peak: number; disconnected: boolean; end: () => void }[] = [];
  const context = {
    state: 'running', currentTime: 10, destination: {},
    createOscillator() {
      const note = { start: 0, stop: 0, peak: 0, disconnected: false, end: () => {} }; notes.push(note);
      const oscillator = { type: '', frequency: { setValueAtTime() {}, exponentialRampToValueAtTime() {} },
        connect() {}, disconnect() { note.disconnected = true; }, onended: null as (() => void) | null,
        start(at: number) { note.start = at; }, stop(at: number) { note.stop = at; } };
      note.end = () => oscillator.onended?.();
      return oscillator;
    },
    createGain() { return { gain: { setValueAtTime() {}, linearRampToValueAtTime(peak: number) { notes[notes.length - 1]!.peak = peak; }, exponentialRampToValueAtTime() {} }, connect() {}, disconnect() {} }; },
  };
  return { context, notes };
}

test('sounds release nodes, stay quiet, and finish within two seconds', () => {
  for (const sound of sounds) {
    const { context, notes } = fakeAudio(); let completions = 0;
    const cancel = playFarmSound(context as unknown as AudioContext,sound,() => completions++);
    assert.ok(notes.length > 0);
    assert.ok(soundNotes(sound).reduce((sum,n) => sum+n.gain,0) <= .2, 'combined levels must remain safe');
    for (const note of notes) {
      assert.ok(note.stop - 10 < 2 && note.stop > note.start);
      assert.ok(note.peak > 0 && note.peak <= .05);
      note.end(); assert.ok(note.disconnected);
    }
    assert.equal(completions,1);
    cancel(); assert.equal(completions,1,'cleanup is idempotent');
    context.state = 'suspended'; const count = notes.length;
    playFarmSound(context as unknown as AudioContext,sound);
    assert.equal(notes.length,count,'never queue sounds while suspended');
  }
});

test('skipping a reveal cancels every scheduled voice immediately', () => {
  const {context,notes} = fakeAudio(); let complete = 0;
  const stop = playFarmSound(context as unknown as AudioContext,'legendary',() => complete++);
  stop(); stop();
  assert.equal(complete,1);
  for (const note of notes) { assert.ok(note.disconnected); assert.equal(note.stop,10); note.end(); }
  assert.equal(complete,1);
});

test('reveal builds pitch and suspense before the rarity-specific finale', () => {
  let previousPitch = 0; let previousDuration = 0;
  for (let i=1;i<=5;i++) {
    const pitch = soundNotes(`hit${i}` as FarmSound)[0]!.frequency;
    assert.ok(pitch > previousPitch); previousPitch = pitch;
	if(i>1) assert.ok(soundNotes(`hit${i}` as FarmSound)[0]!.gain > soundNotes(`hit${i-1}` as FarmSound)[0]!.gain);
  }
  for (const rarity of ['common','uncommon','rare','legendary']) {
    const plan = revealPlan(rarity);
    assert.ok(plan.revealAt > previousDuration); previousDuration = plan.revealAt;
    assert.equal(plan.beats[0],0);
    assert.ok(plan.beats.every((at,i) => i === 0 || at > plan.beats[i-1]!));
	const intervals=plan.beats.slice(1).map((at,i)=>at-plan.beats[i]!);
	assert.ok(intervals.every((gap,i)=>i===0 || gap<intervals[i-1]!), 'strikes must accelerate');
    assert.ok(plan.suspenseAt > plan.beats.at(-1)!);
    assert.ok(plan.revealAt - plan.suspenseAt >= 200);
    assert.ok(plan.beats.length <= 5);
  }
  assert.equal(revealPlan('legendary').sound,'legendary');
  assert.equal(revealPlan('rare').sound,'rare');
  assert.equal(revealPlan('legendary').beats.length,5);
});

test('upgrade reveal accelerates through five strikes before its finale',()=>{
  const plan=gameRevealPlan('upgrade');
  assert.equal(plan.beats.length,5);
  const gaps=plan.beats.slice(1).map((at,i)=>at-plan.beats[i]!);
  assert.ok(gaps.every((gap,i)=>i===0||gap<gaps[i-1]!));
  assert.ok(plan.revealAt-plan.beats.at(-1)!>=700);
  assert.ok(gameRevealPlan('flip').revealAt<plan.revealAt);
});
