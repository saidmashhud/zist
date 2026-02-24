import type { PageLoad } from './$types';
import { getHostBookings } from '$lib/api';

export const load: PageLoad = async ({ fetch }) => {
  const bookings = await getHostBookings(fetch).catch(() => []);
  return { bookings };
};
