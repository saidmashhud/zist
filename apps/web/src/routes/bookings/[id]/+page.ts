import type { PageLoad } from './$types';
import { error } from '@sveltejs/kit';
import { getBooking, getListing } from '$lib/api';

export const load: PageLoad = async ({ fetch, params }) => {
  const booking = await getBooking(fetch, params.id).catch((e: Error) => {
    if (e.message.includes('404')) error(404, 'Booking not found');
    error(500, 'Failed to load booking');
  });

  const listing = await getListing(fetch, booking.listingId).catch(() => null);

  return { booking, listing };
};
