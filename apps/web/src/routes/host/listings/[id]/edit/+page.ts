import type { PageLoad } from './$types';
import { error } from '@sveltejs/kit';
import { getListing } from '$lib/api';

export const load: PageLoad = async ({ fetch, params, url }) => {
  const listing = await getListing(fetch, params.id).catch(() => null);
  if (!listing) error(404, 'Listing not found');
  const created = url.searchParams.has('created');
  return { listing, created };
};
