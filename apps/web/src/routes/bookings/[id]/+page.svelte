<script lang="ts">
  import type { PageData } from './$types';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { nights, fmtDate, fmtAmount } from '$lib/utils';

  let { data }: { data: PageData } = $props();

  const { booking, listing } = $derived(data);

  const n = $derived(nights(booking.checkIn, booking.checkOut));

  // Cancel
  let cancelling = $state(false);
  let cancelError = $state('');
  let cancelled = $state(false);

  async function cancel() {
    if (!confirm('Are you sure you want to cancel this booking?')) return;
    cancelling = true;
    cancelError = '';
    try {
      const res = await fetch(`/api/bookings/${booking.id}/cancel`, { method: 'POST' });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        cancelError = d.error ?? 'Failed to cancel. Please try again.';
      } else {
        cancelled = true;
        location.reload();
      }
    } catch {
      cancelError = 'Network error. Please try again.';
    } finally {
      cancelling = false;
    }
  }

  // Pay now (redirect to existing checkout URL)
  async function payNow() {
    if (!booking.checkoutId) return;
    // checkoutId is the Mashgate checkout session id — redirect to checkout
    const res = await fetch(`/api/checkout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ bookingId: booking.id }),
    });
    if (res.ok) {
      const d = await res.json();
      if (d.checkoutUrl) window.location.href = d.checkoutUrl;
    }
  }

  const canCancel = $derived(
    booking.status === 'pending_host_approval' ||
    booking.status === 'payment_pending' ||
    booking.status === 'confirmed'
  );
</script>

<svelte:head>
  <title>Booking details — Zist</title>
</svelte:head>

<div class="mx-auto max-w-2xl px-6 py-10">

  <!-- Back link -->
  <a href="/bookings" class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800 mb-6">
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
    </svg>
    My trips
  </a>

  <!-- Header -->
  <div class="flex items-start justify-between gap-4 mb-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-900">Booking details</h1>
      <p class="mt-1 text-sm text-gray-400 font-mono">#{booking.id.slice(0, 8)}…</p>
    </div>
    <StatusBadge status={booking.status} />
  </div>

  <!-- Action banners -->
  {#if booking.status === 'payment_pending'}
    <div class="mb-6 rounded-2xl bg-blue-50 border border-blue-200 p-4">
      <p class="font-semibold text-blue-800 text-sm">Complete your payment</p>
      <p class="mt-1 text-sm text-blue-600">
        Your booking was approved! Complete payment to confirm your dates.
        {#if booking.expiresAt}
          Expires {new Date(booking.expiresAt * 1000).toLocaleString()}.
        {/if}
      </p>
      <button
        onclick={payNow}
        class="mt-3 rounded-xl bg-[#ff5a5f] px-5 py-2 text-sm font-semibold text-white hover:bg-[#e84f54] transition-colors"
      >
        Pay now
      </button>
    </div>
  {/if}

  {#if booking.status === 'pending_host_approval'}
    <div class="mb-6 rounded-2xl bg-yellow-50 border border-yellow-200 p-4 text-sm text-yellow-700">
      Your request is with the host. They typically respond within 24 hours.
    </div>
  {/if}

  {#if booking.status === 'confirmed'}
    <div class="mb-6 rounded-2xl bg-green-50 border border-green-200 p-4 text-sm text-green-700">
      Your booking is confirmed. Enjoy your stay!
    </div>
  {/if}

  {#if booking.status === 'rejected'}
    <div class="mb-6 rounded-2xl bg-red-50 border border-red-200 p-4 text-sm text-red-600">
      The host declined your request. Your payment has not been charged.
    </div>
  {/if}

  {#if booking.status === 'failed'}
    <div class="mb-6 rounded-2xl bg-red-50 border border-red-200 p-4 text-sm text-red-600">
      Payment failed. Your dates have been released.
    </div>
  {/if}

  <!-- Listing card -->
  {#if listing}
    <a
      href="/listings/{listing.id}"
      class="mb-6 flex gap-4 rounded-2xl border border-gray-200 p-4 hover:border-gray-300 transition-colors"
    >
      {#if listing.photos && listing.photos.length > 0}
        <img
          src={listing.photos[0].url}
          alt={listing.title}
          class="w-24 h-20 rounded-xl object-cover shrink-0"
        />
      {:else}
        <div class="w-24 h-20 rounded-xl bg-gradient-to-br from-[#ff5a5f]/20 to-[#00a699]/20 shrink-0"></div>
      {/if}
      <div class="min-w-0">
        <p class="font-semibold text-gray-900 leading-tight line-clamp-2">{listing.title}</p>
        <p class="mt-1 text-sm text-gray-500">{listing.city}, {listing.country}</p>
        <p class="mt-0.5 text-xs text-gray-400 capitalize">{listing.type}</p>
      </div>
    </a>
  {/if}

  <!-- Trip details card -->
  <div class="rounded-2xl border border-gray-200 divide-y divide-gray-100 mb-6">
    <div class="px-6 py-4">
      <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Dates</p>
      <p class="font-semibold text-gray-900">{fmtDate(booking.checkIn)} → {fmtDate(booking.checkOut)}</p>
      <p class="text-sm text-gray-500">{n} night{n !== 1 ? 's' : ''}</p>
    </div>
    <div class="px-6 py-4">
      <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Guests</p>
      <p class="font-semibold text-gray-900">{booking.guests} guest{booking.guests !== 1 ? 's' : ''}</p>
    </div>
    <div class="px-6 py-4">
      <p class="text-xs text-gray-400 uppercase tracking-wide mb-2">Price breakdown</p>
      <div class="space-y-1.5 text-sm">
        <div class="flex justify-between text-gray-700">
          <span>Subtotal</span>
          <span>{Number(Number(booking.totalAmount) - Number(booking.platformFee) - Number(booking.cleaningFee)).toLocaleString()} {booking.currency}</span>
        </div>
        {#if Number(booking.cleaningFee) > 0}
          <div class="flex justify-between text-gray-700">
            <span>Cleaning fee</span>
            <span>{Number(booking.cleaningFee).toLocaleString()} {booking.currency}</span>
          </div>
        {/if}
        {#if Number(booking.platformFee) > 0}
          <div class="flex justify-between text-gray-700">
            <span>Service fee</span>
            <span>{Number(booking.platformFee).toLocaleString()} {booking.currency}</span>
          </div>
        {/if}
        <div class="flex justify-between font-semibold text-gray-900 pt-1.5 border-t border-gray-100">
          <span>Total</span>
          <span>{Number(booking.totalAmount).toLocaleString()} {booking.currency}</span>
        </div>
      </div>
    </div>
    <div class="px-6 py-4">
      <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Cancellation policy</p>
      <p class="text-sm text-gray-700 capitalize">{booking.cancellationPolicy}</p>
    </div>
  </div>

  <!-- Cancel -->
  {#if canCancel}
    <div class="rounded-2xl border border-gray-200 px-6 py-5">
      <p class="font-semibold text-gray-900 text-sm">Need to cancel?</p>
      <p class="mt-1 text-sm text-gray-500">
        {#if booking.cancellationPolicy === 'flexible'}
          Free cancellation if cancelled 24+ hours before check-in.
        {:else if booking.cancellationPolicy === 'moderate'}
          Full refund if cancelled 5+ days before check-in, 50% within 1–4 days.
        {:else}
          50% refund if cancelled 14+ days before check-in. No refund within 14 days.
        {/if}
      </p>
      {#if cancelError}
        <p class="mt-2 text-sm text-red-500">{cancelError}</p>
      {/if}
      <button
        onclick={cancel}
        disabled={cancelling}
        class="mt-3 text-sm font-medium text-red-500 hover:text-red-700 hover:underline disabled:opacity-50"
      >
        {cancelling ? 'Cancelling…' : 'Cancel booking'}
      </button>
    </div>
  {/if}

</div>
