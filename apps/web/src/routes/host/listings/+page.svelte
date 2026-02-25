<script lang="ts">
  import type { PageData } from './$types';
  import { fmtAmount } from '$lib/utils';

  let { data }: { data: PageData } = $props();

  async function toggleStatus(listingId: string, current: string) {
    const next = current === 'active' ? 'paused' : 'active';
    await fetch(`/api/listings/${listingId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: next }),
    });
    location.reload();
  }

  async function deleteListing(listingId: string) {
    if (!confirm('Delete this listing? This cannot be undone.')) return;
    await fetch(`/api/listings/${listingId}`, { method: 'DELETE' });
    location.reload();
  }
</script>

<svelte:head>
  <title>My listings — Zist</title>
</svelte:head>

<div class="mx-auto max-w-4xl px-6 py-10">
  <div class="flex items-center justify-between mb-8">
    <div>
      <h1 class="text-2xl font-bold text-gray-900">My listings</h1>
      <p class="text-sm text-gray-500 mt-1">{data.listings.length} listing{data.listings.length !== 1 ? 's' : ''}</p>
    </div>
    <a
      href="/host/listings/new"
      class="rounded-xl bg-[#ff5a5f] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[#e84f54] transition-colors"
    >
      + New listing
    </a>
  </div>

  {#if data.listings.length === 0}
    <div class="rounded-2xl border border-gray-200 px-6 py-16 text-center">
      <p class="text-gray-400 mb-3">You haven't created any listings yet.</p>
      <a href="/host/listings/new" class="text-sm font-medium text-[#ff5a5f] hover:underline">
        Create your first listing
      </a>
    </div>
  {:else}
    <div class="space-y-3">
      {#each data.listings as listing (listing.id)}
        <div class="rounded-2xl border border-gray-200 p-5 hover:border-gray-300 transition-colors">
          <div class="flex gap-4">
            {#if listing.photos && listing.photos.length > 0}
              <img
                src={listing.photos[0].url}
                alt={listing.title}
                class="w-24 h-16 rounded-xl object-cover shrink-0"
              />
            {:else}
              <div class="w-24 h-16 rounded-xl bg-gradient-to-br from-[#ff5a5f]/20 to-[#00a699]/20 shrink-0"></div>
            {/if}
            <div class="min-w-0 flex-1">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <h3 class="text-sm font-semibold text-gray-900 truncate">{listing.title}</h3>
                  <p class="text-xs text-gray-500 mt-0.5">
                    {listing.city} · {Number(listing.pricePerNight).toLocaleString()} {listing.currency}/night
                    · {listing.maxGuests} guest{listing.maxGuests !== 1 ? 's' : ''} max
                  </p>
                </div>
                <span class="shrink-0 text-xs font-medium px-2.5 py-1 rounded-full {
                  listing.status === 'active' ? 'bg-green-50 text-green-700' :
                  listing.status === 'paused' ? 'bg-yellow-50 text-yellow-700' :
                  'bg-gray-100 text-gray-500'
                } capitalize">
                  {listing.status}
                </span>
              </div>
              <div class="flex gap-3 mt-3">
                <a href="/listings/{listing.id}" class="text-xs text-gray-500 hover:text-gray-800">Preview</a>
                <a href="/host/listings/{listing.id}/edit" class="text-xs text-[#ff5a5f] hover:underline">Edit</a>
                <button onclick={() => toggleStatus(listing.id, listing.status)} class="text-xs text-gray-500 hover:text-gray-800">
                  {listing.status === 'active' ? 'Pause' : 'Activate'}
                </button>
                <button onclick={() => deleteListing(listing.id)} class="text-xs text-red-400 hover:text-red-600">
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <div class="mt-6">
    <a href="/host" class="text-sm text-gray-500 hover:text-gray-800">← Back to dashboard</a>
  </div>
</div>
