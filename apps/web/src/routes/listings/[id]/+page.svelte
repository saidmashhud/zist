<script lang="ts">
  import type { PageData } from './$types';
  import type { PricePreview } from '$lib/types';
  import { AMENITIES } from '$lib/types';

  let { data }: { data: PageData } = $props();

  const listing    = $derived(data.listing);
  const user       = $derived((data as any).user as { user_id: string; email: string } | null);
  const blocked    = $derived(
    new Set(
      data.availability
        .filter(d => d.status === 'blocked' || d.status === 'booked')
        .map(d => d.date)
    )
  );

  const today = new Date().toISOString().slice(0, 10);

  // ── Booking widget state ────────────────────────────────────────────────────
  let checkIn   = $state('');
  let checkOut  = $state('');
  let guests    = $state(1);

  let preview       = $state<PricePreview | null>(null);
  let previewLoading = $state(false);
  let previewError  = $state('');

  let submitting  = $state(false);
  let bookingError = $state('');
  let bookingDone  = $state(false); // for request-approval flow
  let bookingId    = $state('');

  // Fetch price preview whenever dates change
  $effect(() => {
    if (checkIn && checkOut && checkOut > checkIn) {
      loadPreview(checkIn, checkOut);
    } else {
      preview = null;
      previewError = '';
    }
  });

  async function loadPreview(ci: string, co: string) {
    previewLoading = true;
    previewError = '';
    try {
      const res = await fetch(
        `/api/listings/${listing.id}/price-preview?check_in=${ci}&check_out=${co}`
      );
      if (!res.ok) {
        const e = await res.json().catch(() => ({}));
        previewError = e.error ?? 'Could not calculate price.';
        preview = null;
      } else {
        preview = await res.json() as PricePreview;
      }
    } catch {
      previewError = 'Could not calculate price.';
      preview = null;
    } finally {
      previewLoading = false;
    }
  }

  async function book() {
    if (!checkIn || !checkOut) { bookingError = 'Select check-in and check-out dates.'; return; }
    if (!preview) { bookingError = 'Wait for price calculation.'; return; }
    bookingError = '';
    submitting = true;

    try {
      const bookRes = await fetch('/api/bookings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          listingId: listing.id,
          checkIn,
          checkOut,
          guests,
          message: '',
        }),
      });

      if (!bookRes.ok) {
        const e = await bookRes.json().catch(() => ({}));
        if (bookRes.status === 409) {
          bookingError = 'Selected dates are no longer available. Please choose different dates.';
        } else {
          bookingError = e.error ?? 'Booking failed.';
        }
        return;
      }

      const booking = await bookRes.json();
      bookingId = booking.id;

      if (listing.instantBook) {
        // Instant book → redirect to Mashgate checkout
        const payRes = await fetch('/api/payments/checkout', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            listingId: listing.id,
            bookingId: booking.id,
            amount: preview!.total,
            currency: preview!.currency,
            successUrl: `${window.location.origin}/bookings?success=1`,
            cancelUrl:  `${window.location.origin}/listings/${listing.id}?cancelled=1`,
            customerEmail: user?.email ?? '',
          }),
        });
        if (!payRes.ok) {
          const e = await payRes.json().catch(() => ({}));
          bookingError = e.error ?? 'Payment session failed.';
          return;
        }
        const { checkoutUrl } = await payRes.json();
        window.location.href = checkoutUrl;
      } else {
        // Request-approval flow → show success
        bookingDone = true;
      }
    } catch {
      bookingError = 'Something went wrong. Please try again.';
    } finally {
      submitting = false;
    }
  }

  // ── Helpers ─────────────────────────────────────────────────────────────────
  const typeLabel: Record<string, string> = {
    apartment: 'Apartment', house: 'House',
    guesthouse: 'Guesthouse', room: 'Private room',
  };

  const amenityMap = Object.fromEntries(AMENITIES.map(a => [a.code, a.label]));

  const amenityIcons: Record<string, string> = {
    wifi: '📶', kitchen: '🍳', parking: '🚗', pool: '🏊', gym: '💪',
    ac: '❄️', heating: '🔥', washer: '🫧', dryer: '🌀', tv: '📺',
    workspace: '💻', balcony: '🏗️', bbq: '🍖', ev_charger: '⚡',
  };

  const policyDesc: Record<string, { label: string; detail: string }> = {
    flexible: {
      label: 'Flexible',
      detail: 'Full refund if cancelled at least 24 hours before check-in.',
    },
    moderate: {
      label: 'Moderate',
      detail:
        'Full refund if cancelled 5+ days before check-in. 50% refund 1–4 days before. No refund within 24 hours.',
    },
    strict: {
      label: 'Strict',
      detail: '50% refund if cancelled 14+ days before check-in. No refund after that.',
    },
  };

  // Photo gallery helpers
  const photos = $derived(listing.photos ?? []);
  const mainPhoto = $derived(photos[0] ?? null);
  const sidePhotos = $derived(photos.slice(1, 5));
</script>

<svelte:head>
  <title>{listing.title} — Zist</title>
</svelte:head>

<div class="mx-auto max-w-6xl px-6 py-8">

  <!-- Back -->
  <a
    href="/listings"
    class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-900 transition-colors mb-6"
  >
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
    </svg>
    All stays
  </a>

  <!-- Title row -->
  <div class="flex flex-wrap items-start justify-between gap-3 mb-4">
    <div>
      <h1 class="text-2xl font-bold text-gray-900 sm:text-3xl">{listing.title}</h1>
      <div class="mt-1.5 flex flex-wrap items-center gap-2 text-sm text-gray-500">
        {#if listing.averageRating > 0}
          <span class="flex items-center gap-1 font-medium text-gray-800">
            <svg class="w-3.5 h-3.5 fill-[#ff5a5f]" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
            </svg>
            {listing.averageRating.toFixed(1)}
          </span>
          <span>·</span>
          <span class="underline cursor-pointer">{listing.reviewCount} review{listing.reviewCount !== 1 ? 's' : ''}</span>
          <span>·</span>
        {/if}
        <span>{listing.city}, {listing.country}</span>
        {#if listing.instantBook}
          <span>·</span>
          <span class="flex items-center gap-1 text-[#ff5a5f] font-medium">
            <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd"/>
            </svg>
            Instant book
          </span>
        {/if}
      </div>
    </div>
  </div>

  <!-- ── Photo gallery ──────────────────────────────────────────────────────── -->
  {#if photos.length === 0}
    <div class="aspect-[16/7] w-full rounded-2xl bg-gradient-to-br from-[#ff5a5f]/15 to-[#00a699]/15 flex items-center justify-center mb-8">
      <svg class="w-20 h-20 text-gray-300" fill="currentColor" viewBox="0 0 24 24">
        <path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/>
      </svg>
    </div>
  {:else if photos.length === 1}
    <div class="aspect-[16/7] w-full rounded-2xl overflow-hidden mb-8">
      <img src={mainPhoto!.url} alt={listing.title} class="w-full h-full object-cover" />
    </div>
  {:else}
    <div class="relative mb-8">
      <div class="grid gap-2 rounded-2xl overflow-hidden {sidePhotos.length >= 2 ? 'grid-cols-[3fr_2fr]' : 'grid-cols-2'}">
        <!-- Main photo -->
        <div class="aspect-[4/3] overflow-hidden {sidePhotos.length >= 2 ? 'row-span-2' : ''}">
          <img src={mainPhoto!.url} alt={listing.title} class="w-full h-full object-cover hover:scale-[1.02] transition-transform duration-300" />
        </div>
        <!-- Side photos -->
        {#each sidePhotos as photo, i}
          <div class="aspect-[4/3] overflow-hidden {i === sidePhotos.length - 1 && photos.length > 5 ? 'relative' : ''}">
            <img src={photo.url} alt={photo.caption || listing.title} class="w-full h-full object-cover hover:scale-[1.02] transition-transform duration-300" />
            {#if i === sidePhotos.length - 1 && photos.length > 5}
              <div class="absolute inset-0 bg-black/40 flex items-center justify-center">
                <span class="text-white font-semibold text-lg">+{photos.length - 5} more</span>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- ── Two-column layout ──────────────────────────────────────────────────── -->
  <div class="grid gap-12 lg:grid-cols-[1fr_380px]">

    <!-- Left: listing details -->
    <div class="min-w-0">

      <!-- Property stats -->
      <div class="flex flex-wrap gap-4 pb-6 border-b border-gray-200">
        <div class="flex items-center gap-2 text-gray-700">
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/>
          </svg>
          <span class="text-sm">{typeLabel[listing.type] ?? listing.type}</span>
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          <span class="text-sm">Up to {listing.maxGuests} guests</span>
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
          </svg>
          <span class="text-sm">{listing.bedrooms} bedroom{listing.bedrooms !== 1 ? 's' : ''}</span>
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"/>
          </svg>
          <span class="text-sm">{listing.beds} bed{listing.beds !== 1 ? 's' : ''}</span>
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
          </svg>
          <span class="text-sm">{listing.bathrooms} bath{listing.bathrooms !== 1 ? 's' : ''}</span>
        </div>
        {#if listing.minNights > 1}
          <div class="flex items-center gap-2 text-gray-700">
            <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707"/>
            </svg>
            <span class="text-sm">Min {listing.minNights} night{listing.minNights !== 1 ? 's' : ''}</span>
          </div>
        {/if}
      </div>

      <!-- About -->
      <section class="py-6 border-b border-gray-200">
        <h2 class="text-xl font-semibold text-gray-900 mb-3">About this place</h2>
        <p class="text-gray-600 leading-relaxed whitespace-pre-line">
          {listing.description || 'No description provided.'}
        </p>
      </section>

      <!-- Amenities -->
      {#if listing.amenities?.length > 0}
        <section class="py-6 border-b border-gray-200">
          <h2 class="text-xl font-semibold text-gray-900 mb-4">What this place offers</h2>
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {#each listing.amenities as code}
              <div class="flex items-center gap-3">
                <span class="text-xl w-7 text-center">{amenityIcons[code] ?? '✓'}</span>
                <span class="text-sm text-gray-700">{amenityMap[code] ?? code}</span>
              </div>
            {/each}
          </div>
        </section>
      {/if}

      <!-- House rules -->
      {#if listing.rules}
        <section class="py-6 border-b border-gray-200">
          <h2 class="text-xl font-semibold text-gray-900 mb-4">House rules</h2>
          <div class="grid gap-3 sm:grid-cols-2">
            {#if listing.rules.checkInFrom}
              <div class="flex items-center gap-3">
                <span class="text-xl">🔑</span>
                <div>
                  <p class="text-xs text-gray-400">Check-in after</p>
                  <p class="text-sm font-medium text-gray-800">{listing.rules.checkInFrom}</p>
                </div>
              </div>
            {/if}
            {#if listing.rules.checkOutBefore}
              <div class="flex items-center gap-3">
                <span class="text-xl">🚪</span>
                <div>
                  <p class="text-xs text-gray-400">Check-out before</p>
                  <p class="text-sm font-medium text-gray-800">{listing.rules.checkOutBefore}</p>
                </div>
              </div>
            {/if}
            {#if listing.rules.quietHoursFrom && listing.rules.quietHoursTo}
              <div class="flex items-center gap-3">
                <span class="text-xl">🌙</span>
                <div>
                  <p class="text-xs text-gray-400">Quiet hours</p>
                  <p class="text-sm font-medium text-gray-800">{listing.rules.quietHoursFrom} – {listing.rules.quietHoursTo}</p>
                </div>
              </div>
            {/if}
            <div class="flex items-center gap-3">
              <span class="text-xl">{listing.rules.smoking ? '🚬' : '🚭'}</span>
              <p class="text-sm text-gray-700">{listing.rules.smoking ? 'Smoking allowed' : 'No smoking'}</p>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-xl">{listing.rules.pets ? '🐾' : '🚫'}</span>
              <p class="text-sm text-gray-700">{listing.rules.pets ? 'Pets allowed' : 'No pets'}</p>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-xl">{listing.rules.parties ? '🎉' : '🚫'}</span>
              <p class="text-sm text-gray-700">{listing.rules.parties ? 'Parties allowed' : 'No parties or events'}</p>
            </div>
          </div>
        </section>
      {/if}

      <!-- Cancellation policy -->
      <section class="py-6">
        <h2 class="text-xl font-semibold text-gray-900 mb-3">Cancellation policy</h2>
        <p class="text-sm font-semibold text-gray-800">
          {policyDesc[listing.cancellationPolicy]?.label ?? listing.cancellationPolicy}
        </p>
        <p class="mt-1 text-sm text-gray-600">
          {policyDesc[listing.cancellationPolicy]?.detail ?? ''}
        </p>
      </section>

    </div>

    <!-- ── Right: booking widget ─────────────────────────────────────────────── -->
    <div>
      <div class="sticky top-24 rounded-2xl border border-gray-200 shadow-lg p-6">

        <!-- Price -->
        <div class="flex items-baseline gap-1.5 mb-5">
          <span class="text-2xl font-bold text-gray-900">{listing.pricePerNight}</span>
          <span class="text-base text-gray-500">{listing.currency} / night</span>
          {#if listing.averageRating > 0}
            <span class="ml-auto flex items-center gap-1 text-sm text-gray-700">
              <svg class="w-3.5 h-3.5 fill-[#ff5a5f]" viewBox="0 0 20 20">
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
              </svg>
              {listing.averageRating.toFixed(1)}
            </span>
          {/if}
        </div>

        {#if !user}
          <!-- Not logged in -->
          <div class="rounded-xl bg-gray-50 p-5 text-center">
            <p class="text-sm text-gray-600 mb-3">Sign in to book this stay</p>
            <a
              href="/api/auth/login"
              class="inline-block w-full rounded-xl bg-[#ff5a5f] px-6 py-3 text-sm font-semibold text-white hover:bg-[#e84f54] transition-colors text-center"
            >
              Sign in to book
            </a>
          </div>

        {:else if bookingDone}
          <!-- Request sent -->
          <div class="rounded-xl bg-green-50 border border-green-200 p-5 text-center">
            <div class="text-3xl mb-2">✅</div>
            <p class="font-semibold text-green-800">Request sent!</p>
            <p class="mt-1 text-sm text-green-700">
              The host will review your request. You'll be notified once they respond.
            </p>
            <a
              href="/bookings"
              class="mt-3 inline-block text-sm font-medium text-green-700 underline"
            >
              View my bookings →
            </a>
          </div>

        {:else}
          <!-- Date pickers -->
          <div class="grid grid-cols-2 rounded-xl border border-gray-300 overflow-hidden mb-3">
            <div class="p-3 border-r border-gray-300">
              <label for="bw-checkin" class="block text-[10px] font-bold uppercase tracking-wider text-gray-500 mb-1">
                Check-in
              </label>
              <input
                id="bw-checkin"
                type="date"
                bind:value={checkIn}
                min={today}
                class="w-full text-sm text-gray-900 bg-transparent focus:outline-none"
              />
            </div>
            <div class="p-3">
              <label for="bw-checkout" class="block text-[10px] font-bold uppercase tracking-wider text-gray-500 mb-1">
                Check-out
              </label>
              <input
                id="bw-checkout"
                type="date"
                bind:value={checkOut}
                min={checkIn || today}
                class="w-full text-sm text-gray-900 bg-transparent focus:outline-none"
              />
            </div>
          </div>

          <!-- Guests -->
          <div class="rounded-xl border border-gray-300 p-3 mb-4">
            <label for="bw-guests" class="block text-[10px] font-bold uppercase tracking-wider text-gray-500 mb-1">
              Guests
            </label>
            <select
              id="bw-guests"
              bind:value={guests}
              class="w-full text-sm text-gray-900 bg-transparent focus:outline-none"
            >
              {#each Array.from({ length: listing.maxGuests }, (_, i) => i + 1) as n}
                <option value={n}>{n} guest{n > 1 ? 's' : ''}</option>
              {/each}
            </select>
          </div>

          <!-- Price breakdown -->
          {#if previewLoading}
            <div class="mb-4 space-y-2">
              <div class="h-4 bg-gray-100 rounded animate-pulse"></div>
              <div class="h-4 bg-gray-100 rounded animate-pulse w-3/4"></div>
              <div class="h-4 bg-gray-100 rounded animate-pulse w-5/6"></div>
            </div>
          {:else if previewError}
            <p class="mb-4 text-sm text-red-500 rounded-lg bg-red-50 px-3 py-2">{previewError}</p>
          {:else if preview}
            <div class="mb-4 space-y-2 text-sm">
              <div class="flex justify-between text-gray-600">
                <span>{preview.pricePerNight} × {preview.nights} night{preview.nights !== 1 ? 's' : ''}</span>
                <span>{preview.subtotal} {preview.currency}</span>
              </div>
              {#if parseFloat(preview.cleaningFee) > 0}
                <div class="flex justify-between text-gray-600">
                  <span>Cleaning fee</span>
                  <span>{preview.cleaningFee} {preview.currency}</span>
                </div>
              {/if}
              <div class="flex justify-between text-gray-600">
                <span>Service fee</span>
                <span>{preview.platformFeeGuest} {preview.currency}</span>
              </div>
              <div class="flex justify-between font-semibold text-gray-900 pt-2 border-t border-gray-200">
                <span>Total</span>
                <span>{preview.total} {preview.currency}</span>
              </div>
            </div>
          {/if}

          <!-- Error -->
          {#if bookingError}
            <p class="mb-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{bookingError}</p>
          {/if}

          <!-- CTA -->
          <button
            type="button"
            onclick={book}
            disabled={submitting || !checkIn || !checkOut || previewLoading}
            class="w-full rounded-xl bg-[#ff5a5f] py-3 text-base font-semibold text-white hover:bg-[#e84f54] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {#if submitting}
              Processing…
            {:else if listing.instantBook}
              Reserve
            {:else}
              Request to book
            {/if}
          </button>

          <p class="mt-2 text-center text-xs text-gray-400">
            {listing.instantBook
              ? "You won't be charged yet — payment via Mashgate"
              : 'Host will confirm within 24 hours'}
          </p>
        {/if}

      </div>
    </div>

  </div>
</div>
