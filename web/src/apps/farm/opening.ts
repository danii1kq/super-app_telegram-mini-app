import type { FarmSound } from './sounds';

export function revealPlan(rarity: string): { beats: number[]; suspenseAt: number; revealAt: number; sound: FarmSound; tier: number } {
  switch (rarity) {
    case 'legendary': return { beats: [0, 760, 1340, 1770, 2050], suspenseAt: 2390, revealAt: 2850, sound: 'legendary', tier: 3 };
    case 'rare': return { beats: [0, 760, 1340, 1770], suspenseAt: 2140, revealAt: 2500, sound: 'rare', tier: 2 };
    case 'uncommon': return { beats: [0, 760, 1340], suspenseAt: 1790, revealAt: 2070, sound: 'harvest', tier: 1 };
    default: return { beats: [0, 760], suspenseAt: 1220, revealAt: 1450, sound: 'harvest', tier: 0 };
  }
}
