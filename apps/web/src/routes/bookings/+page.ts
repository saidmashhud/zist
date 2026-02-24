import type { PageLoad } from './$types';
import { getMyBookings } from '$lib/api';

export const load: PageLoad = async ({ fetch, url }) => {
  const success = url.searchParams.has('success');
  const bookings = await getMyBookings(fetch).catch(() => []);
  return { bookings, success };
};
