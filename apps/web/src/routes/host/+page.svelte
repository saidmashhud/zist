<script lang="ts">
  import type { PageData } from './$types';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { fmtDate, fmtAmount } from '$lib/utils';

  let { data }: { data: PageData } = $props();

  const pendingBookings = $derived(
    data.bookings.filter(b => b.status === 'pending_host_approval')
  );
  const activeBookings = $derived(
    data.bookings.filter(b => b.status === 'confirmed' || b.status === 'payment_pending')
  );

  // Approve/reject
  let actioning = $state<string | null>(null);

  async function approve(bookingId: string) {
    actioning = bookingId;
    await fetch(`/api/bookings/${bookingId}/approve`, { method: 'POST' });
    actioning = null;
    location.reload();
  }

  async function reject(bookingId: string) {
    if (!confirm('Decline this booking request?')) return;
    actioning = bookingId;
    await fetch(`/api/bookings/${bookingId}/reject`, { method: 'POST' });
    actioning = null;
    location.reload();
  }

  // Listing status toggle
  async function toggleStatus(listingId: string, current: string) {
    const next = current === 'active' ? 'paused' : 'active';
    await fetch(`/api/listings/${listingId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: next }),
    });
    location.reload();
  }
</script>

<svelte:head>
  <title>Host dashboard — Zist</title>
</svelte:head>

<div class="mx-auto max-w-5xl px-6 py-10">
  <div class="flex items-center justify-between mb-8">
    <h1 class="text-3xl font-bold text-gray-900">Host dashboard</h1>
    <a
      href="/host/listings/new"
      class="rounded-xl bg-[#ff5a5f] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[#e84f54] transition-colors"
    >
      + New listing
    </a>
  </div>

  <!-- Stats row -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-10">
    <div class="rounded-2xl border border-gray-200 px-5 py-4">
      <p class="text-2xl font-bold text-gray-900">{data.listings.filter(l => l.status === 'active').length}</p>
      <p class="text-xs text-gray-500 mt-0.5">Active listings</p>
    </div>
    <div class="rounded-2xl border border-gray-200 px-5 py-4">
      <p class="text-2xl font-bold text-yellow-600">{pendingBookings.length}</p>
      <p class="text-xs text-gray-500 mt-0.5">Awaiting approval</p>
    </div>
    <div class="rounded-2xl border border-gray-200 px-5 py-4">
      <p class="text-2xl font-bold text-green-600">{activeBookings.length}</p>
      <p class="text-xs text-gray-500 mt-0.5">Confirmed bookings</p>
    </div>
    <div class="rounded-2xl border border-gray-200 px-5 py-4">
      <p class="text-2xl font-bold text-gray-900">{data.listings.length}</p>
      <p class="text-xs text-gray-500 mt-0.5">Total listings</p>
    </div>
  </div>

  <div class="grid lg:grid-cols-5 gap-8">

    <!-- Left: Bookings needing action -->
    <div class="lg:col-span-3">

      {#if pendingBookings.length > 0}
        <h2 class="text-lg font-semibold text-gray-900 mb-3">Action required</h2>
        <div class="space-y-3 mb-8">
          {#each pendingBookings as booking (booking.id)}
            <div class="rounded-2xl border border-yellow-200 bg-yellow-50 p-5">
              <div class="flex items-start justify-between gap-2 mb-3">
                <div>
                  <p class="font-semibold text-gray-900 text-sm">
                    Listing ···{booking.listingId.slice(-6)}
                  </p>
                  <p class="text-xs text-gray-500 mt-0.5">
                    {fmtDate(booking.checkIn)} → {fmtDate(booking.checkOut)}
                    · {booking.guests} guest{booking.guests !== 1 ? 's' : ''}
                    · {fmtAmount(booking.totalAmount, booking.currency)}
                  </p>
                  {#if booking.message}
                    <p class="mt-2 text-sm text-gray-600 italic">"{booking.message}"</p>
                  {/if}
                </div>
              </div>
              <div class="flex gap-2">
                <button
                  onclick={() => approve(booking.id)}
                  disabled={actioning === booking.id}
                  class="flex-1 rounded-xl bg-gray-900 py-2 text-sm font-semibold text-white hover:bg-gray-700 disabled:opacity-50 transition-colors"
                >
                  {actioning === booking.id ? '…' : 'Approve'}
                </button>
                <button
                  onclick={() => reject(booking.id)}
                  disabled={actioning === booking.id}
                  class="flex-1 rounded-xl border border-gray-300 py-2 text-sm font-medium text-gray-700 hover:border-gray-400 disabled:opacity-50 transition-colors"
                >
                  Decline
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}

      <!-- All recent bookings -->
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-lg font-semibold text-gray-900">Recent bookings</h2>
        <a href="/host/bookings" class="text-sm text-[#ff5a5f] hover:underline">View all</a>
      </div>

      {#if data.bookings.length === 0}
        <div class="rounded-2xl border border-gray-200 px-6 py-10 text-center text-sm text-gray-400">
          No bookings yet.
        </div>
      {:else}
        <div class="space-y-2">
          {#each data.bookings.slice(0, 8) as booking (booking.id)}
            <a
              href="/host/bookings/{booking.id}"
              class="flex items-center justify-between gap-4 rounded-xl border border-gray-100 px-4 py-3 hover:border-gray-200 hover:bg-gray-50 transition-all"
            >
              <div class="min-w-0">
                <p class="text-sm font-medium text-gray-900 truncate">
                  ···{booking.listingId.slice(-6)} · {fmtDate(booking.checkIn)} → {fmtDate(booking.checkOut)}
                </p>
                <p class="text-xs text-gray-400 mt-0.5">
                  {booking.guests} guest{booking.guests !== 1 ? 's' : ''}
                  · {fmtAmount(booking.totalAmount, booking.currency)}
                </p>
              </div>
              <StatusBadge status={booking.status} />
            </a>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Right: Listings -->
    <div class="lg:col-span-2">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-lg font-semibold text-gray-900">My listings</h2>
        <a href="/host/listings" class="text-sm text-[#ff5a5f] hover:underline">Manage</a>
      </div>

      {#if data.listings.length === 0}
        <div class="rounded-2xl border border-gray-200 px-6 py-10 text-center">
          <p class="text-sm text-gray-400">No listings yet.</p>
          <a href="/host/listings/new" class="mt-2 inline-block text-sm font-medium text-[#ff5a5f] hover:underline">
            Create your first listing →
          </a>
        </div>
      {:else}
        <div class="space-y-3">
          {#each data.listings as listing (listing.id)}
            <div class="rounded-2xl border border-gray-200 p-4">
              <div class="flex gap-3">
                {#if listing.photos && listing.photos.length > 0}
                  <img
                    src={listing.photos[0].url}
                    alt={listing.title}
                    class="w-16 h-12 rounded-lg object-cover shrink-0"
                  />
                {:else}
                  <div class="w-16 h-12 rounded-lg bg-gradient-to-br from-[#ff5a5f]/20 to-[#00a699]/20 shrink-0"></div>
                {/if}
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-semibold text-gray-900 truncate">{listing.title}</p>
                  <p class="text-xs text-gray-400">{listing.city} · {Number(listing.pricePerNight).toLocaleString()} {listing.currency}/night</p>
                </div>
              </div>
              <div class="mt-3 flex items-center justify-between">
                <span class="text-xs font-medium {listing.status === 'active' ? 'text-green-600' : listing.status === 'paused' ? 'text-yellow-600' : 'text-gray-400'} capitalize">
                  {listing.status}
                </span>
                <div class="flex gap-2">
                  <a href="/listings/{listing.id}" class="text-xs text-gray-500 hover:text-gray-800">
                    View
                  </a>
                  <a href="/host/listings/{listing.id}/edit" class="text-xs text-[#ff5a5f] hover:underline">
                    Edit
                  </a>
                  <button
                    onclick={() => toggleStatus(listing.id, listing.status)}
                    class="text-xs text-gray-500 hover:text-gray-800"
                  >
                    {listing.status === 'active' ? 'Pause' : 'Activate'}
                  </button>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

  </div>
</div>
