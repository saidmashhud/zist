<script lang="ts">
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const statusColor: Record<string, string> = {
    pending:   'bg-yellow-100 text-yellow-700',
    confirmed: 'bg-green-100 text-green-700',
    cancelled: 'bg-red-100 text-red-600'
  };

  async function cancel(bookingId: string) {
    if (!confirm('Cancel this booking?')) return;
    await fetch(`/api/bookings/${bookingId}/cancel`, { method: 'POST' });
    location.reload();
  }
</script>

<svelte:head>
  <title>My trips — Zist</title>
</svelte:head>

<div class="mx-auto max-w-3xl px-6 py-10">
  <h1 class="text-3xl font-bold text-gray-900">My trips</h1>

  {#if data.success}
    <div class="mt-4 rounded-xl bg-green-50 border border-green-200 px-4 py-3 text-green-700 text-sm">
      🎉 Payment successful! Your booking is confirmed.
    </div>
  {/if}

  {#if data.bookings.length === 0}
    <div class="mt-12 text-center">
      <p class="text-gray-500">No trips yet.</p>
      <a href="/listings" class="mt-4 inline-block text-[#ff5a5f] font-medium hover:underline">
        Explore stays →
      </a>
    </div>
  {:else}
    <div class="mt-8 space-y-4">
      {#each data.bookings as booking (booking.id)}
        <div class="rounded-2xl border border-gray-200 p-6">
          <div class="flex items-start justify-between">
            <div>
              <a href="/listings/{booking.listingId}" class="font-semibold text-gray-900 hover:text-[#ff5a5f]">
                Listing {booking.listingId.slice(0, 8)}…
              </a>
              <p class="mt-1 text-sm text-gray-500">
                {booking.checkIn} → {booking.checkOut} · {booking.guests} guest{booking.guests > 1 ? 's' : ''}
              </p>
              <p class="mt-1 text-sm font-semibold text-gray-900">
                {booking.totalAmount} {booking.currency}
              </p>
            </div>
            <span class="rounded-full px-3 py-1 text-xs font-medium {statusColor[booking.status] ?? 'bg-gray-100 text-gray-600'}">
              {booking.status}
            </span>
          </div>

          {#if booking.status === 'pending' || booking.status === 'confirmed'}
            <button
              onclick={() => cancel(booking.id)}
              class="mt-4 text-sm text-red-500 hover:text-red-700 hover:underline"
            >
              Cancel booking
            </button>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
