import type { PageLoad } from './$types';
import type { Listing } from '$lib/types';

export const load: PageLoad = async ({ fetch, url }) => {
  const apiParams = new URLSearchParams();

  const passThrough = ['city', 'check_in', 'check_out', 'guests', 'type',
                       'min_price', 'max_price', 'amenities', 'instant_book'];
  for (const key of passThrough) {
    const val = url.searchParams.get(key);
    if (val) apiParams.set(key, val);
  }
  apiParams.set('limit', '50');

  try {
    const res = await fetch(`/api/listings/search?${apiParams}`);
    if (!res.ok) return { listings: [], filters: Object.fromEntries(url.searchParams) };
    const data = await res.json() as { listings: Listing[]; total: number };
    return {
      listings: data.listings ?? [],
      filters: Object.fromEntries(url.searchParams),
    };
  } catch {
    return { listings: [], filters: Object.fromEntries(url.searchParams) };
  }
};
