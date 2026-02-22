<script lang="ts">
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const { listing } = data;

  // Booking form state
  let checkIn  = $state('');
  let checkOut = $state('');
  let guests   = $state(1);
  let guestId  = $state('guest-' + Math.random().toString(36).slice(2, 9));

  let submitting = $state(false);
  let error      = $state('');

  async function book() {
    if (!checkIn || !checkOut) {
      error = 'Please select check-in and check-out dates.';
      return;
    }
    error = '';
    submitting = true;

    try {
      // 1. Create booking
      const bookRes = await fetch('/api/bookings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          listingId: listing.id,
          guestId,
          checkIn,
          checkOut,
          guests,
          totalAmount: listing.pricePerNight,
          currency: listing.currency
        })
      });

      if (!bookRes.ok) {
        const e = await bookRes.json();
        throw new Error(e.error ?? 'Booking failed');
      }

      const booking = await bookRes.json();

      // 2. Create checkout session via Mashgate
      const payRes = await fetch('/api/payments/checkout', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          listingId: listing.id,
          bookingId: booking.id,
          amount: listing.pricePerNight,
          currency: listing.currency,
          successUrl: `${window.location.origin}/bookings?success=1`,
          cancelUrl: `${window.location.origin}/listings/${listing.id}?cancelled=1`,
          customerEmail: ''
        })
      });

      if (!payRes.ok) {
        const e = await payRes.json();
        throw new Error(e.error ?? 'Payment session failed');
      }

      const { checkoutUrl } = await payRes.json();
      window.location.href = checkoutUrl;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Something went wrong';
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>{listing.title} — Zist</title>
</svelte:head>

<div class="mx-auto max-w-5xl px-6 py-10">
  <!-- Back -->
  <a href="/listings" class="text-sm text-gray-500 hover:text-gray-900 flex items-center gap-1 mb-6">
    ← All stays
  </a>

  <div class="grid gap-10 lg:grid-cols-[1fr_360px]">
    <!-- Left: listing info -->
    <div>
      <!-- Hero image placeholder -->
      <div class="aspect-video w-full rounded-2xl bg-gradient-to-br from-[#ff5a5f]/20 to-[#00a699]/20 flex items-center justify-center text-8xl mb-8">
        🏠
      </div>

      <h1 class="text-3xl font-bold text-gray-900">{listing.title}</h1>
      <p class="mt-1 text-gray-500">{listing.city}, {listing.country}</p>

      <div class="mt-4 flex items-center gap-4 text-sm text-gray-600">
        <span>Up to <strong>{listing.maxGuests}</strong> guests</span>
      </div>

      <hr class="my-6 border-gray-200" />

      <h2 class="text-lg font-semibold text-gray-900">About this place</h2>
      <p class="mt-2 text-gray-600 leading-relaxed whitespace-pre-line">
        {listing.description || 'No description provided.'}
      </p>
    </div>

    <!-- Right: booking card -->
    <div class="h-fit rounded-2xl border border-gray-200 p-6 shadow-md sticky top-24">
      <p class="text-2xl font-bold text-gray-900">
        {listing.pricePerNight} <span class="text-base font-normal text-gray-500">{listing.currency} / night</span>
      </p>

      <form onsubmit={(e) => { e.preventDefault(); book(); }} class="mt-6 space-y-4">
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs font-semibold uppercase tracking-wide text-gray-500 mb-1">
              Check-in
            </label>
            <input
              type="date"
              bind:value={checkIn}
              min={new Date().toISOString().slice(0, 10)}
              required
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-xs font-semibold uppercase tracking-wide text-gray-500 mb-1">
              Check-out
            </label>
            <input
              type="date"
              bind:value={checkOut}
              min={checkIn || new Date().toISOString().slice(0, 10)}
              required
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label class="block text-xs font-semibold uppercase tracking-wide text-gray-500 mb-1">
            Guests
          </label>
          <select
            bind:value={guests}
            class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none"
          >
            {#each Array.from({ length: listing.maxGuests }, (_, i) => i + 1) as n}
              <option value={n}>{n} guest{n > 1 ? 's' : ''}</option>
            {/each}
          </select>
        </div>

        {#if error}
          <p class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{error}</p>
        {/if}

        <button
          type="submit"
          disabled={submitting}
          class="w-full rounded-xl bg-[#ff5a5f] py-3 text-base font-semibold text-white hover:bg-[#e84f54] transition-colors disabled:opacity-60"
        >
          {submitting ? 'Processing…' : 'Reserve & Pay'}
        </button>
      </form>

      <p class="mt-3 text-center text-xs text-gray-400">
        You won't be charged yet — payment via Mashgate
      </p>
    </div>
  </div>
</div>
