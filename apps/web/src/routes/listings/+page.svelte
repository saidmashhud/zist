<script lang="ts">
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  let search = $state('');

  const filtered = $derived(
    data.listings.filter(
      (l) =>
        !search ||
        l.title.toLowerCase().includes(search.toLowerCase()) ||
        l.city.toLowerCase().includes(search.toLowerCase()) ||
        l.country.toLowerCase().includes(search.toLowerCase())
    )
  );
</script>

<svelte:head>
  <title>Explore stays — Zist</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-6 py-10">
  <h1 class="text-3xl font-bold text-gray-900">Explore stays</h1>

  <!-- Search bar -->
  <div class="mt-6 mb-8">
    <input
      type="text"
      placeholder="Search by city or title…"
      bind:value={search}
      class="w-full max-w-md rounded-xl border border-gray-300 px-4 py-3 text-sm shadow-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
    />
  </div>

  {#if filtered.length === 0}
    <p class="text-gray-500">No listings found.</p>
  {:else}
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {#each filtered as listing (listing.id)}
        <a
          href="/listings/{listing.id}"
          class="group block overflow-hidden rounded-2xl border border-gray-100 hover:shadow-lg transition-shadow"
        >
          <div class="aspect-square bg-gradient-to-br from-[#ff5a5f]/15 to-[#00a699]/15 flex items-center justify-center text-5xl">
            🏠
          </div>
          <div class="p-4">
            <p class="font-semibold text-gray-900 group-hover:text-[#ff5a5f] line-clamp-1">
              {listing.title}
            </p>
            <p class="text-sm text-gray-500">{listing.city}, {listing.country}</p>
            <p class="mt-2 text-sm">
              <span class="font-semibold text-gray-900">{listing.pricePerNight} {listing.currency}</span>
              <span class="text-gray-500"> / night</span>
            </p>
            <p class="mt-1 text-xs text-gray-400">Up to {listing.maxGuests} guests</p>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</div>
