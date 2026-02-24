<script lang="ts">
  import { goto } from '$app/navigation';
  import type { PageData } from './$types';
  import ListingCard from '$lib/components/ListingCard.svelte';
  import { AMENITIES } from '$lib/types';

  let { data }: { data: PageData } = $props();

  const today = new Date().toISOString().slice(0, 10);

  // Search bar state — always derived from URL params (reactive to navigation)
  const f = $derived(data.filters as Record<string, string>);

  let city      = $state('');
  let checkIn   = $state('');
  let checkOut  = $state('');
  let guests    = $state(1);
  let propType     = $state('');
  let minPrice     = $state('');
  let maxPrice     = $state('');
  let instantOnly  = $state(false);
  let selectedAmenities = $state<string[]>([]);

  // Sync form state with URL params on every navigation
  $effect(() => {
    city      = f.city ?? '';
    checkIn   = f.check_in ?? '';
    checkOut  = f.check_out ?? '';
    guests    = Number(f.guests ?? 1);
    propType  = f.type ?? '';
    minPrice  = f.min_price ?? '';
    maxPrice  = f.max_price ?? '';
    instantOnly = f.instant_book === 'true';
    selectedAmenities = f.amenities ? f.amenities.split(',').filter(Boolean) : [];
  });

  function buildURL() {
    const p = new URLSearchParams();
    if (city.trim())       p.set('city', city.trim());
    if (checkIn)           p.set('check_in', checkIn);
    if (checkOut)          p.set('check_out', checkOut);
    if (guests > 1)        p.set('guests', String(guests));
    if (propType)          p.set('type', propType);
    if (minPrice)          p.set('min_price', minPrice);
    if (maxPrice)          p.set('max_price', maxPrice);
    if (instantOnly)       p.set('instant_book', 'true');
    if (selectedAmenities.length > 0) p.set('amenities', selectedAmenities.join(','));
    return `/listings?${p}`;
  }

  function applySearch(e: Event) {
    e.preventDefault();
    goto(buildURL());
  }

  function applyFilters() {
    goto(buildURL());
  }

  function toggleAmenity(code: string) {
    if (selectedAmenities.includes(code)) {
      selectedAmenities = selectedAmenities.filter(a => a !== code);
    } else {
      selectedAmenities = [...selectedAmenities, code];
    }
    goto(buildURL());
  }

  const propertyTypes = [
    { value: '', label: 'All types' },
    { value: 'apartment', label: 'Apartment' },
    { value: 'house', label: 'House' },
    { value: 'guesthouse', label: 'Guesthouse' },
    { value: 'room', label: 'Private room' },
  ];

  let showAmenityFilter = $state(false);
  let showPriceFilter   = $state(false);
</script>

<svelte:head>
  <title>{f.city ? `Stays in ${f.city}` : 'Explore stays'} — Zist</title>
</svelte:head>

<!-- Compact search bar -->
<div class="sticky top-[65px] z-40 border-b border-gray-200 bg-white shadow-sm">
  <form
    onsubmit={applySearch}
    class="mx-auto flex max-w-7xl items-center gap-2 px-6 py-3 overflow-x-auto"
  >
    <input
      type="text"
      bind:value={city}
      placeholder="Where?"
      class="min-w-[140px] rounded-full border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
    />
    <input
      type="date"
      bind:value={checkIn}
      min={today}
      class="rounded-full border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
    />
    <input
      type="date"
      bind:value={checkOut}
      min={checkIn || today}
      class="rounded-full border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
    />
    <div class="flex items-center gap-2 rounded-full border border-gray-300 px-4 py-2">
      <button type="button" onclick={() => guests = Math.max(1, guests - 1)} class="text-gray-500 hover:text-gray-800">−</button>
      <span class="text-sm w-4 text-center">{guests}</span>
      <button type="button" onclick={() => guests = Math.min(16, guests + 1)} class="text-gray-500 hover:text-gray-800">+</button>
    </div>
    <button
      type="submit"
      class="shrink-0 rounded-full bg-[#ff5a5f] px-5 py-2 text-sm font-semibold text-white hover:bg-[#e84f54] transition-colors"
    >
      Search
    </button>
  </form>
</div>

<div class="mx-auto max-w-7xl px-6 py-6">

  <!-- Filter bar -->
  <div class="flex flex-wrap items-center gap-2 mb-6">

    <!-- Property type tabs -->
    <div class="flex gap-1 rounded-xl bg-gray-100 p-1">
      {#each propertyTypes as pt}
        <button
          type="button"
          onclick={() => { propType = pt.value; applyFilters(); }}
          class="rounded-lg px-3 py-1.5 text-sm font-medium transition-colors {propType === pt.value ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
        >
          {pt.label}
        </button>
      {/each}
    </div>

    <!-- Price filter -->
    <div class="relative">
      <button
        type="button"
        onclick={() => { showPriceFilter = !showPriceFilter; showAmenityFilter = false; }}
        class="flex items-center gap-1.5 rounded-full border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:border-gray-400 transition-colors {minPrice || maxPrice ? 'border-gray-900 bg-gray-50' : ''}"
      >
        Price
        {#if minPrice || maxPrice}
          <span class="text-xs text-gray-500">
            {minPrice || '0'}–{maxPrice || '∞'}
          </span>
        {/if}
        <svg class="w-3 h-3 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
        </svg>
      </button>

      {#if showPriceFilter}
        <div class="absolute left-0 top-full mt-2 w-64 rounded-2xl border border-gray-200 bg-white p-4 shadow-xl z-50">
          <p class="text-sm font-semibold text-gray-900 mb-3">Price per night</p>
          <div class="flex gap-3">
            <div class="flex-1">
              <label for="price-min" class="text-xs text-gray-500 mb-1 block">Min</label>
              <input
                id="price-min"
                type="number"
                bind:value={minPrice}
                min="0"
                placeholder="0"
                class="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none"
              />
            </div>
            <div class="flex-1">
              <label for="price-max" class="text-xs text-gray-500 mb-1 block">Max</label>
              <input
                id="price-max"
                type="number"
                bind:value={maxPrice}
                min="0"
                placeholder="Any"
                class="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none"
              />
            </div>
          </div>
          <button
            type="button"
            onclick={() => { showPriceFilter = false; applyFilters(); }}
            class="mt-3 w-full rounded-xl bg-gray-900 py-2 text-sm font-semibold text-white hover:bg-gray-700 transition-colors"
          >
            Apply
          </button>
          {#if minPrice || maxPrice}
            <button
              type="button"
              onclick={() => { minPrice = ''; maxPrice = ''; showPriceFilter = false; applyFilters(); }}
              class="mt-2 w-full rounded-xl py-2 text-sm font-medium text-gray-500 hover:text-gray-700 transition-colors"
            >
              Clear
            </button>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Amenities filter -->
    <div class="relative">
      <button
        type="button"
        onclick={() => { showAmenityFilter = !showAmenityFilter; showPriceFilter = false; }}
        class="flex items-center gap-1.5 rounded-full border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:border-gray-400 transition-colors {selectedAmenities.length > 0 ? 'border-gray-900 bg-gray-50' : ''}"
      >
        Amenities
        {#if selectedAmenities.length > 0}
          <span class="rounded-full bg-gray-900 text-white text-xs w-4 h-4 flex items-center justify-center">
            {selectedAmenities.length}
          </span>
        {/if}
        <svg class="w-3 h-3 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
        </svg>
      </button>

      {#if showAmenityFilter}
        <div class="absolute left-0 top-full mt-2 w-72 rounded-2xl border border-gray-200 bg-white p-4 shadow-xl z-50">
          <p class="text-sm font-semibold text-gray-900 mb-3">Amenities</p>
          <div class="grid grid-cols-2 gap-2">
            {#each AMENITIES as amenity}
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={selectedAmenities.includes(amenity.code)}
                  onchange={() => toggleAmenity(amenity.code)}
                  class="rounded border-gray-300 text-[#ff5a5f] focus:ring-[#ff5a5f]"
                />
                <span class="text-sm text-gray-700">{amenity.label}</span>
              </label>
            {/each}
          </div>
          <button
            type="button"
            onclick={() => { showAmenityFilter = false; }}
            class="mt-3 w-full rounded-xl bg-gray-900 py-2 text-sm font-semibold text-white hover:bg-gray-700 transition-colors"
          >
            Done
          </button>
          {#if selectedAmenities.length > 0}
            <button
              type="button"
              onclick={() => { selectedAmenities = []; showAmenityFilter = false; applyFilters(); }}
              class="mt-2 w-full rounded-xl py-2 text-sm font-medium text-gray-500 hover:text-gray-700 transition-colors"
            >
              Clear all
            </button>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Instant book toggle -->
    <button
      type="button"
      onclick={() => { instantOnly = !instantOnly; applyFilters(); }}
      class="flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition-colors {instantOnly ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-300 text-gray-700 hover:border-gray-400'}"
    >
      <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
        <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd"/>
      </svg>
      Instant book
    </button>

    <!-- Active filter chips -->
    {#if f.city}
      <span class="flex items-center gap-1.5 rounded-full bg-[#ff5a5f]/10 px-3 py-1.5 text-sm text-[#ff5a5f]">
        📍 {f.city}
        <a href="/listings" class="hover:text-[#c0393e]">×</a>
      </span>
    {/if}
    {#if f.check_in && f.check_out}
      <span class="flex items-center gap-1 rounded-full bg-gray-100 px-3 py-1.5 text-sm text-gray-600">
        📅 {f.check_in} → {f.check_out}
      </span>
    {/if}
  </div>

  <!-- Results count -->
  <p class="mb-4 text-sm text-gray-500">
    {#if data.listings.length === 0}
      No stays found
    {:else if data.listings.length === 1}
      1 stay found
    {:else}
      {data.listings.length} stays found
    {/if}
    {#if f.city}· in <strong>{f.city}</strong>{/if}
    {#if f.check_in && f.check_out}
      · <strong>{f.check_in}</strong> to <strong>{f.check_out}</strong>
    {/if}
  </p>

  <!-- Results grid -->
  {#if data.listings.length === 0}
    <div class="flex flex-col items-center justify-center py-24 text-center">
      <div class="w-20 h-20 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <svg class="w-10 h-10 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/>
        </svg>
      </div>
      <p class="text-lg font-semibold text-gray-900">No stays found</p>
      <p class="mt-1 text-sm text-gray-500">Try adjusting your dates, location, or filters.</p>
      <a href="/listings" class="mt-4 text-sm font-medium text-[#ff5a5f] hover:underline">
        Clear all filters
      </a>
    </div>
  {:else}
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {#each data.listings as listing (listing.id)}
        <ListingCard {listing} />
      {/each}
    </div>
  {/if}

</div>

<!-- Close dropdowns when clicking outside -->
<svelte:window onclick={(e) => {
  const target = e.target as Element;
  if (!target.closest('[data-filter]')) {
    showPriceFilter = false;
    showAmenityFilter = false;
  }
}} />
