import type { PageLoad } from './$types';
import type { Listing, AvailabilityDay } from '$lib/types';
import { error } from '@sveltejs/kit';

export const load: PageLoad = async ({ fetch, params }) => {
  const res = await fetch(`/api/listings/${params.id}`);
  if (res.status === 404) error(404, 'Listing not found');
  if (!res.ok) error(500, 'Failed to load listing');
  const listing = await res.json() as Listing;

  // Load current month availability so the booking widget can mark unavailable dates.
  const now = new Date();
  const month = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  let availability: AvailabilityDay[] = [];
  try {
    const ar = await fetch(`/api/listings/${params.id}/calendar?month=${month}`);
    if (ar.ok) {
      const ad = await ar.json();
      availability = ad.days ?? [];
    }
  } catch { /* non-fatal */ }

  return { listing, availability };
};
