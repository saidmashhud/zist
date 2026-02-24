<script lang="ts">
  import type { PageData } from './$types';
  import type { BookingStatus } from '$lib/types';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { nights, fmtDate, fmtAmount } from '$lib/utils';

  let { data }: { data: PageData } = $props();

  let actioning = $state<string | null>(null);

  async function approve(id: string) {
    actioning = id;
    await fetch(`/api/bookings/${id}/approve`, { method: 'POST' });
    actioning = null;
    location.reload();
  }

  async function reject(id: string) {
    if (!confirm('Decline this booking?')) return;
    actioning = id;
    await fetch(`/api/bookings/${id}/reject`, { method: 'POST' });
    actioning = null;
    location.reload();
  }

  // Filter
  let filter = $state<BookingStatus | 'all'>('all');
  const filters: Array<{ value: BookingStatus | 'all'; label: string }> = [
    { value: 'all', label: 'All' },
    { value: 'pending_host_approval', label: 'Awaiting approval' },
    { value: 'confirmed', label: 'Confirmed' },
    { value: 'payment_pending', label: 'Payment pending' },
    { value: 'completed', label: 'Completed' },
    { value: 'cancelled_by_guest', label: 'Cancelled' },
  ];

  const filtered = $derived(
    filter === 'all' ? data.bookings : data.bookings.filter(b => b.status === filter)
  );
</script>

<svelte:head>
  <title>Host bookings — Zist</title>
</svelte:head>

<div class="mx-auto max-w-4xl px-6 py-10">
  <div class="flex items-center gap-4 mb-6">
    <a href="/host" class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
      </svg>
      Dashboard
    </a>
    <h1 class="text-2xl font-bold text-gray-900">All bookings</h1>
  </div>

  <!-- Filter tabs -->
  <div class="flex flex-wrap gap-2 mb-6">
    {#each filters as f}
      <button
        type="button"
        onclick={() => filter = f.value}
        class="rounded-full px-4 py-1.5 text-sm font-medium transition-colors {filter === f.value ? 'bg-gray-900 text-white' : 'border border-gray-300 text-gray-600 hover:border-gray-400'}"
      >
        {f.label}
        {#if f.value !== 'all'}
          <span class="ml-1 text-xs opacity-70">
            ({data.bookings.filter(b => b.status === f.value).length})
          </span>
        {/if}
      </button>
    {/each}
  </div>

  {#if filtered.length === 0}
    <div class="rounded-2xl border border-gray-200 px-6 py-16 text-center text-sm text-gray-400">
      No bookings match this filter.
    </div>
  {:else}
    <div class="space-y-3">
      {#each filtered as booking (booking.id)}
        <div class="rounded-2xl border border-gray-200 p-5 {booking.status === 'pending_host_approval' ? 'border-yellow-200 bg-yellow-50' : ''}">
          <div class="flex items-start justify-between gap-4">
            <div>
              <a href="/listings/{booking.listingId}" class="text-sm font-semibold text-gray-900 hover:text-[#ff5a5f]">
                Listing ···{booking.listingId.slice(-6)}
              </a>
              <p class="mt-0.5 text-sm text-gray-500">
                {fmtDate(booking.checkIn)} → {fmtDate(booking.checkOut)}
                · {nights(booking.checkIn, booking.checkOut)} night{nights(booking.checkIn, booking.checkOut) !== 1 ? 's' : ''}
                · {booking.guests} guest{booking.guests !== 1 ? 's' : ''}
              </p>
              <p class="mt-0.5 text-sm font-semibold text-gray-900">
                {fmtAmount(booking.totalAmount, booking.currency)}
              </p>
              {#if booking.message}
                <p class="mt-1.5 text-sm text-gray-500 italic">"{booking.message}"</p>
              {/if}
            </div>
            <StatusBadge status={booking.status} />
          </div>

          {#if booking.status === 'pending_host_approval'}
            <div class="mt-3 flex gap-2">
              <button
                onclick={() => approve(booking.id)}
                disabled={actioning === booking.id}
                class="rounded-xl bg-gray-900 px-5 py-2 text-sm font-semibold text-white hover:bg-gray-700 disabled:opacity-50 transition-colors"
              >
                {actioning === booking.id ? '…' : 'Approve'}
              </button>
              <button
                onclick={() => reject(booking.id)}
                disabled={actioning === booking.id}
                class="rounded-xl border border-gray-300 px-5 py-2 text-sm font-medium text-gray-700 hover:border-gray-400 disabled:opacity-50 transition-colors"
              >
                Decline
              </button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
