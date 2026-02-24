import type { Booking } from '$lib/types';

/** Fetch the authenticated guest's own bookings. */
export async function getMyBookings(fetch: typeof globalThis.fetch): Promise<Booking[]> {
  const res = await fetch('/api/bookings');
  if (!res.ok) throw new Error(`bookings: ${res.status}`);
  const data = await res.json();
  return data.bookings ?? [];
}

/** Fetch a single booking by ID. Throws if not found. */
export async function getBooking(fetch: typeof globalThis.fetch, id: string): Promise<Booking> {
  const res = await fetch(`/api/bookings/${id}`);
  if (!res.ok) throw new Error(`bookings/${id}: ${res.status}`);
  return res.json();
}

/** Fetch all bookings on the authenticated host's listings. */
export async function getHostBookings(fetch: typeof globalThis.fetch): Promise<Booking[]> {
  const res = await fetch('/api/bookings/host');
  if (!res.ok) throw new Error(`bookings/host: ${res.status}`);
  const data = await res.json();
  return data.bookings ?? [];
}

export interface CreateBookingInput {
  listingId: string;
  checkIn: string;
  checkOut: string;
  guests: number;
  message?: string;
}

/** Create a new booking request. */
export async function createBooking(
  fetch: typeof globalThis.fetch,
  input: CreateBookingInput
): Promise<Booking> {
  const res = await fetch('/api/bookings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error ?? `createBooking: ${res.status}`);
  }
  return res.json();
}

/** Cancel a booking (guest or host). */
export async function cancelBooking(fetch: typeof globalThis.fetch, id: string): Promise<void> {
  const res = await fetch(`/api/bookings/${id}/cancel`, { method: 'POST' });
  if (!res.ok) throw new Error(`cancel ${id}: ${res.status}`);
}

/** Host: approve a booking. */
export async function approveBooking(fetch: typeof globalThis.fetch, id: string): Promise<void> {
  const res = await fetch(`/api/bookings/${id}/approve`, { method: 'POST' });
  if (!res.ok) throw new Error(`approve ${id}: ${res.status}`);
}

/** Host: reject a booking. */
export async function rejectBooking(fetch: typeof globalThis.fetch, id: string): Promise<void> {
  const res = await fetch(`/api/bookings/${id}/reject`, { method: 'POST' });
  if (!res.ok) throw new Error(`reject ${id}: ${res.status}`);
}
