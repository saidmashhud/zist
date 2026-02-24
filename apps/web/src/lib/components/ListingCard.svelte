<script lang="ts">
  import type { Listing } from '$lib/types';

  let { listing }: { listing: Listing } = $props();

  const typeLabel: Record<string, string> = {
    apartment: 'Apartment',
    house: 'House',
    guesthouse: 'Guesthouse',
    room: 'Private room',
  };
</script>

<a
  href="/listings/{listing.id}"
  class="group block overflow-hidden rounded-2xl hover:shadow-lg transition-all duration-200"
>
  <!-- Photo -->
  <div class="relative aspect-[4/3] overflow-hidden rounded-2xl bg-gray-100">
    {#if listing.photos?.[0]?.url}
      <img
        src={listing.photos[0].url}
        alt={listing.title}
        class="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
      />
    {:else}
      <div class="h-full w-full bg-gradient-to-br from-[#ff5a5f]/20 to-[#00a699]/20 flex items-center justify-center">
        <svg class="w-12 h-12 text-gray-300" fill="currentColor" viewBox="0 0 24 24">
          <path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/>
        </svg>
      </div>
    {/if}
    {#if listing.instantBook}
      <span class="absolute top-3 left-3 rounded-full bg-white/90 px-2.5 py-0.5 text-xs font-semibold text-gray-800 shadow-sm backdrop-blur-sm">
        Instant book
      </span>
    {/if}
  </div>

  <!-- Info -->
  <div class="mt-3 px-0.5">
    <div class="flex items-start justify-between gap-2">
      <p class="font-semibold text-gray-900 line-clamp-1 group-hover:text-[#ff5a5f] transition-colors">
        {listing.title}
      </p>
      {#if listing.averageRating > 0}
        <span class="shrink-0 flex items-center gap-0.5 text-sm text-gray-700">
          <svg class="w-3.5 h-3.5 fill-current text-[#ff5a5f]" viewBox="0 0 20 20">
            <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
          </svg>
          {listing.averageRating.toFixed(1)}
          {#if listing.reviewCount > 0}
            <span class="text-gray-400 text-xs">({listing.reviewCount})</span>
          {/if}
        </span>
      {/if}
    </div>
    <p class="mt-0.5 text-sm text-gray-500">
      {listing.city}, {listing.country}
      <span class="text-gray-300"> · </span>
      {typeLabel[listing.type] ?? listing.type}
    </p>
    <p class="mt-0.5 text-sm text-gray-400">
      {listing.bedrooms} bed{listing.bedrooms !== 1 ? 's' : ''}
      · up to {listing.maxGuests} guests
    </p>
    <p class="mt-2 text-sm">
      <span class="font-semibold text-gray-900">{listing.pricePerNight} {listing.currency}</span>
      <span class="text-gray-400"> / night</span>
    </p>
  </div>
</a>
