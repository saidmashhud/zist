<script lang="ts">
  import type { PageData } from './$types';
  import ListingCard from '$lib/components/ListingCard.svelte';

  let { data }: { data: PageData } = $props();

  let city = $state('');
  let checkIn = $state('');
  let checkOut = $state('');
  let guests = $state(1);

  const today = new Date().toISOString().slice(0, 10);

  function search(e: Event) {
    e.preventDefault();
    const params = new URLSearchParams();
    if (city.trim()) params.set('city', city.trim());
    if (checkIn) params.set('check_in', checkIn);
    if (checkOut) params.set('check_out', checkOut);
    if (guests > 1) params.set('guests', String(guests));
    window.location.href = `/listings?${params}`;
  }

  const popularCities = [
    'Tashkent', 'Samarkand', 'Bukhara', 'Almaty', 'Astana',
    'Bishkek', 'Dushanbe', 'Ashgabat', 'Khiva', 'Namangan',
  ];
</script>

<svelte:head>
  <title>Zist — Find your stay in Central Asia</title>
</svelte:head>

<!-- Hero -->
<section class="relative overflow-hidden bg-gradient-to-br from-[#ff5a5f]/8 to-[#00a699]/8 py-20 px-6">
  <div class="mx-auto max-w-4xl text-center">
    <h1 class="text-5xl font-bold tracking-tight text-gray-900 sm:text-6xl">
      Find your stay<br />in Central Asia
    </h1>
    <p class="mt-4 text-lg text-gray-500 max-w-xl mx-auto">
      Discover unique homes in Uzbekistan, Kazakhstan, Kyrgyzstan, Tajikistan and beyond.
    </p>

    <!-- Search card -->
    <form
      onsubmit={search}
      class="mt-10 flex flex-col sm:flex-row items-stretch sm:items-center gap-0 rounded-2xl bg-white shadow-xl ring-1 ring-gray-200 overflow-hidden text-left"
    >
      <!-- Destination -->
      <div class="flex-1 min-w-0 border-b sm:border-b-0 sm:border-r border-gray-200">
        <label for="search-city" class="block px-5 pt-3 text-xs font-bold uppercase tracking-wider text-gray-500">
          Where
        </label>
        <input
          id="search-city"
          list="cities"
          type="text"
          bind:value={city}
          placeholder="City or destination"
          class="w-full px-5 pb-3 text-sm text-gray-900 placeholder-gray-400 bg-transparent focus:outline-none"
        />
        <datalist id="cities">
          {#each popularCities as c}
            <option value={c}></option>
          {/each}
        </datalist>
      </div>

      <!-- Check-in -->
      <div class="flex-1 min-w-0 border-b sm:border-b-0 sm:border-r border-gray-200">
        <label for="search-checkin" class="block px-5 pt-3 text-xs font-bold uppercase tracking-wider text-gray-500">
          Check-in
        </label>
        <input
          id="search-checkin"
          type="date"
          bind:value={checkIn}
          min={today}
          class="w-full px-5 pb-3 text-sm text-gray-900 bg-transparent focus:outline-none"
        />
      </div>

      <!-- Check-out -->
      <div class="flex-1 min-w-0 border-b sm:border-b-0 sm:border-r border-gray-200">
        <label for="search-checkout" class="block px-5 pt-3 text-xs font-bold uppercase tracking-wider text-gray-500">
          Check-out
        </label>
        <input
          id="search-checkout"
          type="date"
          bind:value={checkOut}
          min={checkIn || today}
          class="w-full px-5 pb-3 text-sm text-gray-900 bg-transparent focus:outline-none"
        />
      </div>

      <!-- Guests -->
      <div class="flex-shrink-0 border-b sm:border-b-0 sm:border-r border-gray-200">
        <p class="px-5 pt-3 text-xs font-bold uppercase tracking-wider text-gray-500">
          Guests
        </p>
        <div class="flex items-center gap-3 px-5 pb-3">
          <button
            type="button"
            onclick={() => guests = Math.max(1, guests - 1)}
            class="w-6 h-6 rounded-full border border-gray-300 flex items-center justify-center text-gray-600 hover:border-gray-400 transition-colors text-lg leading-none"
          >−</button>
          <span class="w-4 text-center text-sm font-medium text-gray-900">{guests}</span>
          <button
            type="button"
            onclick={() => guests = Math.min(16, guests + 1)}
            class="w-6 h-6 rounded-full border border-gray-300 flex items-center justify-center text-gray-600 hover:border-gray-400 transition-colors text-lg leading-none"
          >+</button>
        </div>
      </div>

      <!-- Submit -->
      <div class="px-4 py-3 sm:py-0 flex items-center justify-center">
        <button
          type="submit"
          class="w-full sm:w-auto rounded-xl bg-[#ff5a5f] px-6 py-3 text-sm font-semibold text-white hover:bg-[#e84f54] transition-colors shadow-sm"
        >
          Search
        </button>
      </div>
    </form>
  </div>
</section>

<!-- Popular destinations -->
<section class="mx-auto max-w-7xl px-6 pt-14 pb-4">
  <h2 class="text-lg font-semibold text-gray-900 mb-4">Popular destinations</h2>
  <div class="flex flex-wrap gap-2">
    {#each popularCities.slice(0, 6) as c}
      <a
        href="/listings?city={c}"
        class="rounded-full border border-gray-200 bg-white px-4 py-2 text-sm text-gray-700 hover:border-[#ff5a5f] hover:text-[#ff5a5f] transition-colors shadow-sm"
      >
        {c}
      </a>
    {/each}
  </div>
</section>

<!-- Featured listings -->
{#if data.listings.length > 0}
  <section class="mx-auto max-w-7xl px-6 py-10">
    <h2 class="mb-6 text-2xl font-bold text-gray-900">Featured stays</h2>
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {#each data.listings as listing (listing.id)}
        <ListingCard {listing} />
      {/each}
    </div>
  </section>
{/if}
