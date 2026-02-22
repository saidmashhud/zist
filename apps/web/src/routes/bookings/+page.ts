import type { PageLoad } from './$types';
import type { Booking } from '$lib/types';

export const load: PageLoad = async ({ fetch, url }) => {
  const guestId = url.searchParams.get('guestId') ?? '';
  const success  = url.searchParams.has('success');

  const path = guestId ? `/api/bookings?guestId=${encodeURIComponent(guestId)}` : '/api/bookings';
  const res = await fetch(path);
  if (!res.ok) return { bookings: [], success };
  const data = await res.json() as { bookings: Booking[] };
  return { bookings: data.bookings ?? [], success };
};
