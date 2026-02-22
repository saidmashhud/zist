<script lang="ts">
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();
</script>

<svelte:head>
  <title>Zist — Find your stay in Central Asia</title>
</svelte:head>

<!-- Hero -->
<section class="relative bg-gradient-to-br from-[#ff5a5f]/10 to-[#00a699]/10 py-24">
  <div class="mx-auto max-w-3xl px-6 text-center">
    <h1 class="text-5xl font-bold tracking-tight text-gray-900">
      Find a place to stay<br />across Central Asia
    </h1>
    <p class="mt-4 text-xl text-gray-600">
      Discover unique homes, apartments, and guesthouses in Uzbekistan, Kazakhstan, and beyond.
    </p>
    <a
      href="/listings"
      class="mt-8 inline-block rounded-xl bg-[#ff5a5f] px-8 py-4 text-lg font-semibold text-white shadow-lg hover:bg-[#e84f54] transition-colors"
    >
      Explore stays
    </a>
  </div>
</section>

<!-- Featured listings -->
<section class="mx-auto max-w-7xl px-6 py-16">
  <h2 class="mb-8 text-2xl font-bold text-gray-900">Featured stays</h2>

  {#if data.listings.length === 0}
    <p class="text-gray-500">No listings yet. Be the first to add one!</p>
  {:else}
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {#each data.listings as listing (listing.id)}
        <a href="/listings/{listing.id}" class="group block rounded-2xl overflow-hidden hover:shadow-md transition-shadow">
          <!-- Placeholder image -->
          <div class="aspect-square bg-gradient-to-br from-[#ff5a5f]/20 to-[#00a699]/20 flex items-center justify-center">
            <span class="text-4xl">🏠</span>
          </div>
          <div class="p-4">
            <div class="flex items-start justify-between">
              <div>
                <p class="font-semibold text-gray-900 group-hover:text-[#ff5a5f] line-clamp-1">
                  {listing.title}
                </p>
                <p class="text-sm text-gray-500">{listing.city}, {listing.country}</p>
              </div>
            </div>
            <p class="mt-2 text-sm font-semibold text-gray-900">
              {listing.pricePerNight} {listing.currency}
              <span class="font-normal text-gray-500">/ night</span>
            </p>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</section>
