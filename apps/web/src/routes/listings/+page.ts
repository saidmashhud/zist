import type { PageLoad } from './$types';
import type { Listing } from '$lib/types';

export const load: PageLoad = async ({ fetch }) => {
  const res = await fetch('/api/listings');
  if (!res.ok) return { listings: [] };
  const data = await res.json() as { listings: Listing[] };
  return { listings: data.listings ?? [] };
};
