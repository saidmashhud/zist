import type { PageLoad } from './$types';
import type { Listing } from '$lib/types';
import { error } from '@sveltejs/kit';

export const load: PageLoad = async ({ fetch, params }) => {
  const res = await fetch(`/api/listings/${params.id}`);
  if (res.status === 404) error(404, 'Listing not found');
  if (!res.ok) error(500, 'Failed to load listing');
  const listing = await res.json() as Listing;
  return { listing };
};
