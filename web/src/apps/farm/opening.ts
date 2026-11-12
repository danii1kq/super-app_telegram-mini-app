import type { FarmSound } from './sounds';

export function revealPlan(rarity: string): { beats: number[]; suspenseAt: number; revealAt: number; sound: FarmSound; tier: number } {
  switch (rarity) {
    case 'legendary': return { beats: [0, 620, 1100, 1470, 1750], suspenseAt: 2130, revealAt: 2580, sound: 'legendary', tier: 3 };
    case 'rare': return { beats: [0, 620, 1100, 1470], suspenseAt: 1830, revealAt: 2200, sound: 'rare', tier: 2 };
    case 'uncommon': return { beats: [0, 620, 1100], suspenseAt: 1450, revealAt: 1730, sound: 'harvest', tier: 1 };
    default: return { beats: [0, 620], suspenseAt: 970, revealAt: 1220, sound: 'harvest', tier: 0 };
  }
}

export function gameRevealPlan(kind:string):{beats:number[];revealAt:number}{
  return kind==='upgrade'?{beats:[0,520,940,1280,1510],revealAt:2350}:{beats:[0,430,780],revealAt:1350};
}
