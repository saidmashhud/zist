import type { PageLoad } from './$types';
import { getMyListings, getHostBookings } from '$lib/api';

export const load: PageLoad = async ({ fetch }) => {
  const [listings, bookings] = await Promise.all([
    getMyListings(fetch).catch(() => []),
    getHostBookings(fetch).catch(() => []),
  ]);
  return { listings, bookings };
};
