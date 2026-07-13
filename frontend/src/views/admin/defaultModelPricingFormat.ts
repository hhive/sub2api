export function formatTokenPrice(value: number | null): string {
  if (value === null) return '-'
  if (value === 0) return '$0 / 1M'
  return `$${(value * 1_000_000).toPrecision(6)} / 1M`
}

export function formatImagePrice(value: number | null): string {
  if (value === null) return '-'
  if (value === 0) return '$0 / image'
  return `$${value.toPrecision(6)} / image`
}
