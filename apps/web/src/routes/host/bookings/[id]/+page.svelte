<script lang="ts">
  import type { PageData } from './$types';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { nights as calcNights, fmtDate } from '$lib/utils';

  let { data }: { data: PageData } = $props();
  const { booking, listing } = $derived(data);

  const nights = $derived(calcNights(booking.checkIn, booking.checkOut));

  let actioning = $state(false);
  let actionError = $state('');

  async function approve() {
    actioning = true;
    actionError = '';
    const res = await fetch(`/api/bookings/${booking.id}/approve`, { method: 'POST' });
    if (!res.ok) actionError = 'Failed to approve.';
    else location.reload();
    actioning = false;
  }

  async function reject() {
    if (!confirm('Decline this booking request?')) return;
    actioning = true;
    actionError = '';
    const res = await fetch(`/api/bookings/${booking.id}/reject`, { method: 'POST' });
    if (!res.ok) actionError = 'Failed to decline.';
    else location.reload();
    actioning = false;
  }

  async function cancelAsHost() {
    if (!confirm('Cancel this booking? You may need to issue a full refund.')) return;
    actioning = true;
    actionError = '';
    const res = await fetch(`/api/bookings/${booking.id}/cancel`, { method: 'POST' });
    if (!res.ok) actionError = 'Failed to cancel.';
    else location.reload();
    actioning = false;
  }
</script>

<svelte:head>
  <title>Booking — Zist Host</title>
</svelte:head>

<div class="mx-auto max-w-2xl px-6 py-10">
  <a href="/host/bookings" class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800 mb-6">
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
    </svg>
    All bookings
  </a>

  <div class="flex items-start justify-between gap-4 mb-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-900">Booking details</h1>
      <p class="mt-1 text-sm text-gray-400 font-mono">#{booking.id.slice(0, 8)}…</p>
    </div>
    <StatusBadge status={booking.status} />
  </div>

  <!-- Action area for pending approval -->
  {#if booking.status === 'pending_host_approval'}
    <div class="mb-6 rounded-2xl bg-yellow-50 border border-yellow-200 p-5">
      <p class="font-semibold text-yellow-800 text-sm mb-1">Guest request</p>
      {#if booking.message}
        <p class="text-sm text-yellow-700 italic mb-3">"{booking.message}"</p>
      {/if}
      <p class="text-sm text-yellow-600 mb-3">
        {booking.guests} guest{booking.guests !== 1 ? 's' : ''}
        · {nights} night{nights !== 1 ? 's' : ''}
        · {Number(booking.totalAmount).toLocaleString()} {booking.currency}
      </p>
      {#if actionError}<p class="text-sm text-red-500 mb-2">{actionError}</p>{/if}
      <div class="flex gap-2">
        <button onclick={approve} disabled={actioning} class="flex-1 rounded-xl bg-gray-900 py-2 text-sm font-semibold text-white hover:bg-gray-700 disabled:opacity-50 transition-colors">
          {actioning ? '…' : 'Approve'}
        </button>
        <button onclick={reject} disabled={actioning} class="flex-1 rounded-xl border border-gray-300 py-2 text-sm font-medium text-gray-700 hover:border-gray-400 disabled:opacity-50 transition-colors">
          Decline
        </button>
      </div>
    </div>
  {/if}

  <!-- Listing -->
  {#if listing}
    <a href="/listings/{listing.id}" class="mb-6 flex gap-4 rounded-2xl border border-gray-200 p-4 hover:border-gray-300 transition-colors">
      {#if listing.photos && listing.photos.length > 0}
        <img src={listing.photos[0].url} alt={listing.title} class="w-24 h-20 rounded-xl object-cover shrink-0" />
      {:else}
        <div class="w-24 h-20 rounded-xl bg-gradient-to-br from-[#ff5a5f]/20 to-[#00a699]/20 shrink-0"></div>
      {/if}
      <div class="min-w-0">
        <p class="font-semibold text-gray-900 leading-tight line-clamp-2">{listing.title}</p>
        <p class="mt-1 text-sm text-gray-500">{listing.city}, {listing.country}</p>
      </div>
    </a>
  {/if}

  <!-- Details -->
  <div class="rounded-2xl border border-gray-200 divide-y divide-gray-100 mb-6">
    <div class="px-6 py-4">
      <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Dates</p>
      <p class="font-semibold text-gray-900">{fmtDate(booking.checkIn)} → {fmtDate(booking.checkOut)}</p>
      <p class="text-sm text-gray-500">{nights} night{nights !== 1 ? 's' : ''}</p>
    </div>
    <div class="px-6 py-4">
      <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Guests</p>
      <p class="font-semibold text-gray-900">{booking.guests} guest{booking.guests !== 1 ? 's' : ''}</p>
    </div>
    <div class="px-6 py-4">
      <p class="text-xs text-gray-400 uppercase tracking-wide mb-2">Earnings</p>
      <div class="space-y-1.5 text-sm">
        <div class="flex justify-between text-gray-700">
          <span>Subtotal (guest pays)</span>
          <span>{Number(booking.totalAmount).toLocaleString()} {booking.currency}</span>
        </div>
        <div class="flex justify-between text-gray-500">
          <span>Platform service fee</span>
          <span>− {Number(booking.platformFee).toLocaleString()} {booking.currency}</span>
        </div>
        <div class="flex justify-between font-semibold text-gray-900 pt-1.5 border-t border-gray-100">
          <span>Your earnings</span>
          <span>{(Number(booking.totalAmount) - Number(booking.platformFee)).toLocaleString()} {booking.currency}</span>
        </div>
      </div>
    </div>
  </div>

  <!-- Cancel as host -->
  {#if booking.status === 'confirmed' || booking.status === 'payment_pending'}
    <div class="rounded-2xl border border-gray-200 px-6 py-5">
      <p class="font-semibold text-gray-900 text-sm">Need to cancel?</p>
      <p class="mt-1 text-sm text-gray-500">
        Cancelling a confirmed booking will issue a full refund to the guest and may impact your hosting status.
      </p>
      {#if actionError}<p class="mt-2 text-sm text-red-500">{actionError}</p>{/if}
      <button
        onclick={cancelAsHost}
        disabled={actioning}
        class="mt-3 text-sm font-medium text-red-500 hover:text-red-700 hover:underline disabled:opacity-50"
      >
        {actioning ? 'Cancelling…' : 'Cancel booking'}
      </button>
    </div>
  {/if}
</div>
