import type { PageLoad } from './$types';
import type { Booking } from '$lib/types';

export const load: PageLoad = async ({ fetch, url }) => {
  const success = url.searchParams.has('success');

  // Bookings are filtered server-side by the authenticated user's ID (via X-User-ID header).
  // No guestId query param needed.
  const res = await fetch('/api/bookings');
  if (!res.ok) return { bookings: [], success };
  const data = await res.json() as { bookings: Booking[] };
  return { bookings: data.bookings ?? [], success };
};
