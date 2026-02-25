import type { PageLoad } from './$types';
import { getMyListings } from '$lib/api';

export const load: PageLoad = async ({ fetch }) => {
  const listings = await getMyListings(fetch).catch(() => []);
  return { listings };
};
