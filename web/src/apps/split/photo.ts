export interface Crop { x: number; y: number; width: number; height: number }

export function cropPixels(width: number, height: number, crop: Crop) {
  if (![width, height, crop.x, crop.y, crop.width, crop.height].every(Number.isFinite) || width <= 0 || height <= 0) throw new Error('invalid crop');
  const x = Math.min(width - 1, Math.max(0, Math.round(width * crop.x / 100)));
  const y = Math.min(height - 1, Math.max(0, Math.round(height * crop.y / 100)));
  return { x, y, width: Math.max(1, Math.min(width - x, Math.round(width * crop.width / 100))), height: Math.max(1, Math.min(height - y, Math.round(height * crop.height / 100))) };
}

export function rotatedSize(width: number, height: number, turns: number) {
  const scale = Math.min(1, 4096 / Math.max(width, height));
  const w = Math.max(1, Math.round(width * scale)), h = Math.max(1, Math.round(height * scale));
  return turns % 2 ? { width: h, height: w } : { width: w, height: h };
}
