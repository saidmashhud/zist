import type { PageLoad } from './$types';
import type { Listing } from '$lib/types';
import { error } from '@sveltejs/kit';

export interface Review {
  id: string;
  bookingId: string;
  listingId: string;
  guestId: string;
  rating: number;
  comment: string;
  reply: string;
  createdAt: number;
}

export const load: PageLoad = async ({ fetch, params, url }) => {
  const [listingRes, reviewsRes] = await Promise.all([
    fetch(`/api/listings/${params.id}`),
    fetch(`/api/reviews/listing/${params.id}?limit=50`),
  ]);

  if (listingRes.status === 404) error(404, 'Listing not found');
  if (!listingRes.ok) error(500, 'Failed to load listing');

  const listing = await listingRes.json() as Listing;
  const reviews: Review[] = reviewsRes.ok ? ((await reviewsRes.json()).reviews ?? []) : [];

  // bookingId + hostId are passed when navigating from a completed booking detail page
  const bookingId = url.searchParams.get('bookingId') ?? '';
  const hostId    = url.searchParams.get('hostId') ?? '';

  return { listing, reviews, bookingId, hostId };
};
