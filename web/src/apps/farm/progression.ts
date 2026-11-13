import type { Crop, Pack, Upgrade } from './api';

export function upcomingUnlocks(level: number, crops: Crop[], packs: Pack[], upgrades: Upgrade[], seedLevels: Record<string,number>, packLevels: Record<string,number>) {
  const entries = [
    ...crops.map(crop => ({ level: seedLevels[crop.id] ?? 1, name: crop.name })),
    ...packs.map(pack => ({ level: packLevels[pack.id] ?? 1, name: pack.name })),
    ...upgrades.filter(upgrade => upgrade.level < upgrade.max_level).map(upgrade => ({ level: upgrade.required_level, name: `${upgrade.name} ${upgrade.level + 1}` })),
  ].filter(entry => entry.level > level).sort((a,b) => a.level - b.level);
  const levels = [...new Set(entries.map(entry => entry.level))].slice(0,3);
  return levels.map(required => ({ level: required, names: [...new Set(entries.filter(entry => entry.level === required).map(entry => entry.name))] }));
}
