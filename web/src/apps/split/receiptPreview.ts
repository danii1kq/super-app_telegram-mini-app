export const MAX_RECEIPT_MINOR = 1_000_000_000_000;
export const MAX_RECEIPT_ITEMS = 200;

export interface ReceiptPreviewItem { key: string; amount: number; units: Record<string, number> }

/** Match server integer division, including discounts and numeric user-id ordering. */
function weighted(amount: bigint, entries: [string, bigint][]): Record<string, bigint> {
  const ordered = entries.filter(([, weight]) => weight > 0n).sort((a, b) => a[1] === b[1] ? BigInt(a[0]) < BigInt(b[0]) ? -1 : 1 : a[1] > b[1] ? -1 : 1);
  const total = ordered.reduce((sum, [, weight]) => sum + weight, 0n);
  if (!total) return {};
  const result: Record<string, bigint> = {};
  let assigned = 0n;
  for (const [id, weight] of ordered) { const value = amount * weight / total; result[id] = value; assigned += value; }
  let remainder = amount - assigned;
  for (let index = 0; remainder && index < ordered.length; index++) {
    const entry = ordered[index]; if (!entry) break;
    const step = remainder > 0n ? 1n : -1n;
    result[entry[0]] = (result[entry[0]] ?? 0n) + step; remainder -= step;
  }
  return result;
}

export function receiptPreview(items: ReceiptPreviewItem[], discount: number, service: number, total: number, payerId: string) {
  const amounts = [discount, service, total, ...items.map((item) => item.amount)];
  let valid = items.length <= MAX_RECEIPT_ITEMS && amounts.every((value) => Number.isSafeInteger(value) && value >= 0 && value <= MAX_RECEIPT_MINOR);
  const unallocated: string[] = [];
  const shares: Record<string, bigint> = {};
  if (valid) for (const item of items) {
    const units = Object.entries(item.units);
    if (!units.length) { unallocated.push(item.key); continue; }
    if (units.some(([id, count]) => !/^[1-9]\d{0,18}$/.test(id) || !Number.isInteger(count) || count < 1 || count > 999)) { valid = false; continue; }
    const part = weighted(BigInt(item.amount), units.map(([id, count]) => [id, BigInt(count)]));
    for (const [id, amount] of Object.entries(part)) shares[id] = (shares[id] ?? 0n) + amount;
  }
  const initial = Object.values(shares).reduce((sum, amount) => sum + amount, 0n);
  if (valid && (BigInt(discount) > initial || initial > BigInt(MAX_RECEIPT_MINOR) || initial + BigInt(service) - BigInt(discount) > BigInt(MAX_RECEIPT_MINOR))) valid = false;
  if (valid) {
    const adjustment = weighted(BigInt(service) - BigInt(discount), Object.entries(shares));
    for (const [id, amount] of Object.entries(adjustment)) shares[id] = (shares[id] ?? 0n) + amount;
  }
  const perPerson = Object.fromEntries(Object.entries(shares).map(([id, amount]) => [id, Number(amount)]));
  const sum = Object.values(perPerson).reduce((value, amount) => value + amount, 0);
  const owes = Object.fromEntries(Object.entries(perPerson).filter(([id, amount]) => id !== payerId && amount > 0));
  return { perPerson, owes, sum, difference: total - sum, unallocated, valid };
}
