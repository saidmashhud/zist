// Shared formatting utilities used across multiple routes.

/** Number of nights between two YYYY-MM-DD date strings. */
export function nights(checkIn: string, checkOut: string): number {
  const ms = new Date(checkOut).getTime() - new Date(checkIn).getTime();
  return Math.round(ms / 86_400_000);
}

/** Format a YYYY-MM-DD date as a human-readable string (e.g. "Jan 15, 2025"). */
export function fmtDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

/** Format a numeric amount string with thousands separators. */
export function fmtAmount(amount: string, currency: string): string {
  return `${Number(amount).toLocaleString()} ${currency}`;
}
