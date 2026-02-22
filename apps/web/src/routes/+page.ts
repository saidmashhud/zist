import type { PageLoad } from './$types';
import type { Listing } from '$lib/types';

export const load: PageLoad = async ({ fetch }) => {
  try {
    const res = await fetch('/api/listings');
    if (!res.ok) return { listings: [] };
    const data = await res.json() as { listings: Listing[] };
    // Show up to 8 featured listings on the home page
    return { listings: (data.listings ?? []).slice(0, 8) };
  } catch {
    return { listings: [] };
  }
};
