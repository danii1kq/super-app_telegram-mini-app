export type FarmSound = 'plant' | 'harvest' | 'sell' | 'buy' | 'pack' | 'rare' | 'legendary' | 'upgrade' | 'tap' | 'hit1' | 'hit2' | 'hit3' | 'hit4' | 'hit5';

export interface SoundNote { frequency: number; at: number; duration: number; gain: number; type?: OscillatorType; endFrequency?: number }

export function soundNotes(sound: FarmSound): SoundNote[] {
  if (sound.startsWith('hit')) {
    const step = Number(sound.slice(3)) - 1;
    const frequency = [330, 440, 587, 784, 1047][step]!;
    // Inharmonic overtones give each rising strike a bell-like "ting".
    return [
      { frequency, at: 0, duration: .5, gain: .024 + step * .006 },
      { frequency: frequency * 2.76, at: .002, duration: .24, gain: .014 },
      { frequency: frequency * 4.08, at: .003, duration: .12, gain: .006 },
      { frequency: 85, endFrequency: 42, at: 0, duration: .15, gain: .025, type: 'triangle' },
    ];
  }
  if (sound === 'legendary' || sound === 'rare') {
    const legendary = sound === 'legendary';
    return [
      { frequency: legendary ? 65 : 98, endFrequency: 40, at: 0, duration: .35, gain: .04, type: 'triangle' },
      ...[523, 659, 784, 1047].map((frequency,i) => ({ frequency, at: i * .065, duration: legendary ? 1.15 : .65, gain: .023 })),
      { frequency: legendary ? 2093 : 1568, at: .22, duration: legendary ? 1.3 : .7, gain: .012 },
    ];
  }
  const melodies: Record<Exclude<FarmSound, 'rare' | 'legendary' | 'hit1' | 'hit2' | 'hit3' | 'hit4' | 'hit5'>, number[]> = {
    plant: [262, 392], harvest: [523, 659, 784], sell: [784, 1047, 1319],
    buy: [440, 587], pack: [262, 392], upgrade: [330, 440, 554, 659], tap: [587],
  };
  return melodies[sound as keyof typeof melodies].map((frequency,i) => ({ frequency, at: i * .075, duration: .25, gain: .03 }));
}

export function playFarmSound(context: AudioContext, sound: FarmSound, onComplete?: () => void): () => void {
  if (context.state !== 'running') return () => {};
  const notes = soundNotes(sound);
  let remaining = notes.length;
  const active = notes.map(note => {
    const start = context.currentTime + note.at;
    const oscillator = context.createOscillator();
    const envelope = context.createGain();
    oscillator.type = note.type ?? 'sine';
    oscillator.frequency.setValueAtTime(note.frequency, start);
    oscillator.frequency.exponentialRampToValueAtTime(note.endFrequency ?? note.frequency * .997, start + note.duration);
    envelope.gain.setValueAtTime(0, start);
    envelope.gain.linearRampToValueAtTime(note.gain, start + .006);
    envelope.gain.exponentialRampToValueAtTime(.0001, start + note.duration);
    oscillator.connect(envelope); envelope.connect(context.destination);
    let ended = false;
    const cleanup = () => {
      if (ended) return;
      ended = true; oscillator.disconnect(); envelope.disconnect();
      remaining--; if (remaining === 0) onComplete?.();
    };
    oscillator.onended = cleanup;
    oscillator.start(start); oscillator.stop(start + note.duration + .02);
    return () => { try { oscillator.stop(context.currentTime); } catch { /* Already ended. */ } cleanup(); };
  });
  return () => active.forEach(stop => stop());
}
