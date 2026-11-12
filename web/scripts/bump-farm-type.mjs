// Raises the farm UI type scale. The stylesheet hardcodes px sizes, and most of
// the interface sits at 8-11px, which is unreadable on a laptop and worse on a
// phone. This rewrites font-size values through a floor map, leaving display
// sizes (headings) untouched.
import { readFileSync, writeFileSync } from 'node:fs';

const files = [
  'src/apps/farm/farm.css',
  'src/apps/farm/collection.css',
  'src/apps/farm/themes.css',
  'src/apps/farm/mutations.css',
  'src/apps/farm/worlds.css',
];

// Anything below 14px moves up; 14px and above is already legible.
const map = new Map([
  [4, 8], [5, 8], [6, 9], [7, 10], [8, 10],
  [9, 11], [10, 12], [11, 13], [12, 13], [13, 14],
]);

let total = 0;
const report = [];

for (const file of files) {
  const before = readFileSync(file, 'utf8');
  let changed = 0;
  const after = before.replace(/font-size:\s*(\d+(?:\.\d+)?)px/g, (whole, raw) => {
    const size = Number(raw);
    const next = map.get(size);
    if (next === undefined) return whole;
    changed++;
    return `font-size:${next}px`;
  });
  if (changed > 0) {
    writeFileSync(file, after, 'utf8');
    total += changed;
    report.push(`${file}: ${changed}`);
  }
}

console.log(report.join('\n'));
console.log(`total rewritten: ${total}`);
