<script lang="ts">
  import type { PageData } from './$types';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { nights, fmtDate, fmtAmount } from '$lib/utils';

  let { data }: { data: PageData } = $props();

  const active = $derived(
    data.bookings.filter(b =>
      b.status === 'pending_host_approval' || b.status === 'payment_pending' || b.status === 'confirmed'
    )
  );
  const past = $derived(
    data.bookings.filter(b =>
      b.status !== 'pending_host_approval' && b.status !== 'payment_pending' && b.status !== 'confirmed'
    )
  );
</script>

<svelte:head>
  <title>My trips — Zist</title>
</svelte:head>

<div class="mx-auto max-w-3xl px-6 py-10">
  <h1 class="text-3xl font-bold text-gray-900">My trips</h1>

  {#if data.success}
    <div class="mt-4 rounded-xl bg-green-50 border border-green-200 px-4 py-3 text-green-700 text-sm">
      Your booking is confirmed. Enjoy your stay!
    </div>
  {/if}

  {#if data.bookings.length === 0}
    <div class="mt-16 flex flex-col items-center text-center">
      <div class="w-20 h-20 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <svg class="w-10 h-10 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
        </svg>
      </div>
      <p class="text-lg font-semibold text-gray-900">No trips yet</p>
      <p class="mt-1 text-sm text-gray-500">Book a stay and it'll show up here.</p>
      <a href="/listings" class="mt-4 inline-block text-sm font-semibold text-[#ff5a5f] hover:underline">
        Explore stays →
      </a>
    </div>

  {:else}

    <!-- Active bookings -->
    {#if active.length > 0}
      <h2 class="mt-8 text-lg font-semibold text-gray-900">Upcoming & active</h2>
      <div class="mt-3 space-y-4">
        {#each active as booking (booking.id)}
          <a
            href="/bookings/{booking.id}"
            class="block rounded-2xl border border-gray-200 p-6 hover:border-gray-300 hover:shadow-sm transition-all"
          >
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0">
                <p class="font-semibold text-gray-900 truncate">
                  Listing ···{booking.listingId.slice(-6)}
                </p>
                <p class="mt-1 text-sm text-gray-500">
                  {fmtDate(booking.checkIn)} → {fmtDate(booking.checkOut)}
                  · {nights(booking.checkIn, booking.checkOut)} night{nights(booking.checkIn, booking.checkOut) !== 1 ? 's' : ''}
                  · {booking.guests} guest{booking.guests !== 1 ? 's' : ''}
                </p>
                <p class="mt-1 text-sm font-semibold text-gray-900">
                  {fmtAmount(booking.totalAmount, booking.currency)}
                </p>
              </div>
              <StatusBadge status={booking.status} />
            </div>

            {#if booking.status === 'payment_pending'}
              <div class="mt-3 flex items-center gap-2 rounded-xl bg-blue-50 px-4 py-2.5 text-sm text-blue-700">
                <svg class="w-4 h-4 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/>
                </svg>
                Action required — complete payment to confirm your booking.
                <span class="ml-auto font-semibold text-[#ff5a5f]">Pay now →</span>
              </div>
            {/if}

            {#if booking.status === 'pending_host_approval'}
              <p class="mt-3 text-xs text-gray-400">
                Waiting for host to approve your request.
              </p>
            {/if}
          </a>
        {/each}
      </div>
    {/if}

    <!-- Past bookings -->
    {#if past.length > 0}
      <h2 class="mt-10 text-lg font-semibold text-gray-900">Past & cancelled</h2>
      <div class="mt-3 space-y-3">
        {#each past as booking (booking.id)}
          <a
            href="/bookings/{booking.id}"
            class="block rounded-2xl border border-gray-200 px-6 py-4 hover:border-gray-300 transition-all opacity-80 hover:opacity-100"
          >
            <div class="flex items-center justify-between gap-4">
              <div class="min-w-0">
                <p class="font-medium text-gray-900 truncate text-sm">
                  Listing ···{booking.listingId.slice(-6)}
                </p>
                <p class="mt-0.5 text-xs text-gray-400">
                  {fmtDate(booking.checkIn)} → {fmtDate(booking.checkOut)}
                  · {fmtAmount(booking.totalAmount, booking.currency)}
                </p>
              </div>
              <StatusBadge status={booking.status} />
            </div>
          </a>
        {/each}
      </div>
    {/if}

  {/if}
</div>
