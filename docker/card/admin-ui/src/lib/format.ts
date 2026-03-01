const satFormatter = new Intl.NumberFormat("en-US");

export function formatSats(sats: number): string {
  return satFormatter.format(sats) + " sats";
}

export function formatTimestamp(unix: number): string {
  return new Date(unix * 1000).toLocaleString();
}
