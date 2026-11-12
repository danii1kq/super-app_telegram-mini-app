export type FarmSound = 'plant' | 'harvest' | 'sell' | 'buy' | 'pack' | 'rare' | 'legendary' | 'upgrade' | 'tap';

const melodies: Record<FarmSound, number[]> = {
  plant: [262, 392], harvest: [523, 659, 784], sell: [784, 1047, 1319],
  buy: [440, 587], pack: [392, 494, 587, 784], rare: [523, 659, 784, 1047, 1319],
  legendary: [262, 392, 523, 659, 784, 1047, 1319, 1568],
  upgrade: [330, 440, 554, 659], tap: [587],
};

// Soft, short synthesized notes need no downloaded audio or tracking requests.
export function playFarmSound(context: AudioContext, sound: FarmSound) {
  if (context.state !== 'running') return;
  melodies[sound].forEach((frequency, index) => {
    const start = context.currentTime + index * .075;
    const oscillator = context.createOscillator();
    const envelope = context.createGain();
    oscillator.type = 'sine';
    oscillator.frequency.setValueAtTime(frequency, start);
    oscillator.frequency.exponentialRampToValueAtTime(frequency * .98, start + .2);
    envelope.gain.setValueAtTime(0, start);
    envelope.gain.linearRampToValueAtTime(.035, start + .012);
    envelope.gain.exponentialRampToValueAtTime(.0001, start + .23);
    oscillator.connect(envelope);
    envelope.connect(context.destination);
    oscillator.onended = () => { oscillator.disconnect(); envelope.disconnect(); };
    oscillator.start(start);
    oscillator.stop(start + .25);
  });
}
