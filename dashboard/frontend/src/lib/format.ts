export function formatMilli(m: number): string {
  return m >= 1000 ? `${(m / 1000).toFixed(2)} cores` : `${Math.round(m)}m`;
}

export function formatBytes(bytes: number): string {
  const units = ["B", "Ki", "Mi", "Gi", "Ti"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value.toFixed(1)}${units[unit]}`;
}
