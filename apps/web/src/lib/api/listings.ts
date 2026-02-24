import type { Listing, SearchFilters, PricePreview, Photo } from '$lib/types';

/** Fetch a single listing by ID. Returns null if not found. */
export async function getListing(fetch: typeof globalThis.fetch, id: string): Promise<Listing | null> {
  const res = await fetch(`/api/listings/${id}`);
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`listings/${id}: ${res.status}`);
  return res.json();
}

/** List the authenticated host's own listings. */
export async function getMyListings(fetch: typeof globalThis.fetch): Promise<Listing[]> {
  const res = await fetch('/api/listings/mine');
  if (!res.ok) throw new Error(`listings/mine: ${res.status}`);
  const data = await res.json();
  return data.listings ?? [];
}

/** Search listings with optional filters. */
export async function searchListings(
  fetch: typeof globalThis.fetch,
  filters: SearchFilters = {}
): Promise<Listing[]> {
  const params = new URLSearchParams();
  if (filters.city)        params.set('city', filters.city);
  if (filters.checkIn)     params.set('checkIn', filters.checkIn);
  if (filters.checkOut)    params.set('checkOut', filters.checkOut);
  if (filters.guests)      params.set('guests', String(filters.guests));
  if (filters.type)        params.set('type', filters.type);
  if (filters.minPrice)    params.set('minPrice', filters.minPrice);
  if (filters.maxPrice)    params.set('maxPrice', filters.maxPrice);
  if (filters.instantBook) params.set('instantBook', 'true');

  const res = await fetch(`/api/listings/search?${params}`);
  if (!res.ok) throw new Error(`listings/search: ${res.status}`);
  const data = await res.json();
  return data.listings ?? [];
}

/** Fetch price preview for date range. */
export async function getPricePreview(
  fetch: typeof globalThis.fetch,
  listingId: string,
  checkIn: string,
  checkOut: string,
  guests: number
): Promise<PricePreview> {
  const params = new URLSearchParams({ checkIn, checkOut, guests: String(guests) });
  const res = await fetch(`/api/listings/${listingId}/price-preview?${params}`);
  if (!res.ok) throw new Error(`price-preview: ${res.status}`);
  return res.json();
}

/** Fetch photos for a listing. */
export async function getPhotos(fetch: typeof globalThis.fetch, listingId: string): Promise<Photo[]> {
  const res = await fetch(`/api/listings/${listingId}/photos`);
  if (!res.ok) return [];
  const data = await res.json();
  return data.photos ?? [];
}
