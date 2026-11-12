import { useCallback, useEffect, useRef, useState } from 'react';
import { deviceStorage } from '@/shared/telegram/sdk';
import { playFarmSound, type FarmSound } from './sounds';

export function useFarmSound() {
  const [enabled, setEnabled] = useState(() => deviceStorage.get('srunal.farm.sound') !== 'off');
  const context = useRef<AudioContext | null>(null);
  const active = useRef(new Set<() => void>());
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  // Called only inside a click, so browsers can unlock audio without autoplay.
  const prime = useCallback(() => {
    if (!enabledRef.current || typeof AudioContext === 'undefined') return;
    try {
      context.current ??= new AudioContext();
      if (context.current.state === 'suspended') void context.current.resume().catch(() => {});
    } catch { /* Audio is optional on clients without a usable audio device. */ }
  }, []);

  const play = useCallback((sound: FarmSound) => {
    if (!enabledRef.current || !context.current || context.current.state !== 'running' || document.hidden) return;
    try {
      let cancel = () => {};
      cancel = playFarmSound(context.current, sound, () => active.current.delete(cancel));
      active.current.add(cancel);
    } catch { /* Never block a game action on audio. */ }
  }, []);

  const stop = useCallback(() => { active.current.forEach(cancel => cancel()); active.current.clear(); }, []);

  const toggle = () => {
    const next = !enabled;
    enabledRef.current = next;
    setEnabled(next);
    deviceStorage.set('srunal.farm.sound', next ? 'on' : 'off');
    if (next) { prime(); if (context.current) void context.current.resume().then(() => play('tap')).catch(() => {}); }
    else if (context.current) { stop(); void context.current.close().catch(() => {}); context.current = null; }
  };

  useEffect(() => () => { stop(); if (context.current) void context.current.close().catch(() => {}); }, [stop]);
  return { enabled, prime, play, stop, toggle };
}
