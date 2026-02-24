<script lang="ts">
  import { goto } from '$app/navigation';
  import { AMENITIES } from '$lib/types';

  // Form state
  let title       = $state('');
  let description = $state('');
  let city        = $state('');
  let country     = $state('');
  let address     = $state('');
  let type        = $state('apartment');
  let bedrooms    = $state(1);
  let beds        = $state(1);
  let bathrooms   = $state(1);
  let maxGuests   = $state(2);
  let pricePerNight = $state('');
  let currency      = $state('USD');
  let cleaningFee   = $state('0');
  let minNights     = $state(1);
  let maxNights     = $state(30);
  let cancellationPolicy = $state('flexible');
  let instantBook   = $state(false);
  let selectedAmenities = $state<string[]>([]);

  // Rules
  let checkInFrom     = $state('14:00');
  let checkOutBefore  = $state('12:00');
  let smoking  = $state(false);
  let pets     = $state(false);
  let parties  = $state(false);

  let submitting = $state(false);
  let error      = $state('');
  let step       = $state(1); // multi-step: 1=basics, 2=details, 3=pricing

  function toggleAmenity(code: string) {
    if (selectedAmenities.includes(code)) {
      selectedAmenities = selectedAmenities.filter(a => a !== code);
    } else {
      selectedAmenities = [...selectedAmenities, code];
    }
  }

  async function submit() {
    submitting = true;
    error = '';
    try {
      const res = await fetch('/api/listings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title,
          description,
          city,
          country,
          address,
          type,
          bedrooms,
          beds,
          bathrooms,
          maxGuests,
          pricePerNight,
          currency,
          cleaningFee,
          minNights,
          maxNights,
          cancellationPolicy,
          instantBook,
          amenities: selectedAmenities,
          rules: { checkInFrom, checkOutBefore, smoking, pets, parties },
        }),
      });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        error = d.error ?? 'Failed to create listing.';
        return;
      }
      const listing = await res.json();
      goto(`/host/listings/${listing.id}/edit?created=1`);
    } catch {
      error = 'Network error. Please try again.';
    } finally {
      submitting = false;
    }
  }

  const propertyTypes = [
    { value: 'apartment',  label: 'Apartment' },
    { value: 'house',      label: 'House' },
    { value: 'guesthouse', label: 'Guesthouse' },
    { value: 'room',       label: 'Private room' },
  ];

  const currencies = ['USD', 'UZS', 'KZT', 'EUR'];
</script>

<svelte:head>
  <title>Create listing — Zist</title>
</svelte:head>

<div class="mx-auto max-w-2xl px-6 py-10">
  <a href="/host" class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800 mb-6">
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
    </svg>
    Dashboard
  </a>

  <h1 class="text-2xl font-bold text-gray-900 mb-2">Create a new listing</h1>
  <p class="text-sm text-gray-500 mb-8">Fill in the details to list your property on Zist.</p>

  <!-- Step indicator -->
  <div class="flex gap-2 mb-8">
    {#each [1, 2, 3] as s}
      <div class="flex items-center gap-2">
        <div class="w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold {step >= s ? 'bg-[#ff5a5f] text-white' : 'bg-gray-100 text-gray-400'}">
          {s}
        </div>
        {#if s < 3}
          <div class="w-8 h-px {step > s ? 'bg-[#ff5a5f]' : 'bg-gray-200'}"></div>
        {/if}
      </div>
    {/each}
    <div class="ml-2 text-xs text-gray-400 self-center">
      {step === 1 ? 'Location & type' : step === 2 ? 'Details & amenities' : 'Pricing & rules'}
    </div>
  </div>

  {#if error}
    <div class="mb-4 rounded-xl bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-600">{error}</div>
  {/if}

  <!-- Step 1: Location & type -->
  {#if step === 1}
    <div class="space-y-5">
      <div>
        <label for="title" class="block text-sm font-medium text-gray-700 mb-1">Listing title</label>
        <input
          id="title"
          type="text"
          bind:value={title}
          placeholder="Cozy apartment in the heart of Tashkent"
          class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
        />
      </div>

      <div>
        <label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          id="description"
          bind:value={description}
          rows="4"
          placeholder="Describe your place…"
          class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f] resize-none"
        ></textarea>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="city" class="block text-sm font-medium text-gray-700 mb-1">City</label>
          <input
            id="city"
            type="text"
            bind:value={city}
            placeholder="Tashkent"
            class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
          />
        </div>
        <div>
          <label for="country" class="block text-sm font-medium text-gray-700 mb-1">Country</label>
          <input
            id="country"
            type="text"
            bind:value={country}
            placeholder="Uzbekistan"
            class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
          />
        </div>
      </div>

      <div>
        <label for="address" class="block text-sm font-medium text-gray-700 mb-1">Address</label>
        <input
          id="address"
          type="text"
          bind:value={address}
          placeholder="Street address"
          class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
        />
      </div>

      <div>
        <p class="text-sm font-medium text-gray-700 mb-2">Property type</p>
        <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {#each propertyTypes as pt}
            <button
              type="button"
              onclick={() => type = pt.value}
              class="rounded-xl border px-3 py-3 text-sm font-medium text-center transition-colors {type === pt.value ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-200 text-gray-700 hover:border-gray-400'}"
            >
              {pt.label}
            </button>
          {/each}
        </div>
      </div>
    </div>

    <div class="mt-8 flex justify-end">
      <button
        type="button"
        onclick={() => step = 2}
        disabled={!title || !city || !country}
        class="rounded-xl bg-gray-900 px-6 py-2.5 text-sm font-semibold text-white hover:bg-gray-700 disabled:opacity-40 transition-colors"
      >
        Continue →
      </button>
    </div>

  <!-- Step 2: Details & amenities -->
  {:else if step === 2}
    <div class="space-y-5">
      <div>
        <p class="text-sm font-medium text-gray-700 mb-3">Property details</p>
        <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
          {#each [
            { label: 'Bedrooms', bind: 'bedrooms', val: bedrooms, set: (v: number) => bedrooms = v },
            { label: 'Beds', bind: 'beds', val: beds, set: (v: number) => beds = v },
            { label: 'Bathrooms', bind: 'bathrooms', val: bathrooms, set: (v: number) => bathrooms = v },
            { label: 'Max guests', bind: 'guests', val: maxGuests, set: (v: number) => maxGuests = v },
          ] as field}
            <div class="rounded-xl border border-gray-200 p-3 text-center">
              <p class="text-xs text-gray-500 mb-2">{field.label}</p>
              <div class="flex items-center justify-center gap-3">
                <button type="button" onclick={() => field.set(Math.max(1, field.val - 1))} class="w-6 h-6 rounded-full border border-gray-300 text-gray-600 hover:border-gray-400 text-sm leading-none">−</button>
                <span class="text-sm font-semibold w-4 text-center">{field.val}</span>
                <button type="button" onclick={() => field.set(Math.min(20, field.val + 1))} class="w-6 h-6 rounded-full border border-gray-300 text-gray-600 hover:border-gray-400 text-sm leading-none">+</button>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <div>
        <p class="text-sm font-medium text-gray-700 mb-3">Amenities</p>
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {#each AMENITIES as amenity}
            <label class="flex items-center gap-2 rounded-xl border border-gray-200 px-3 py-2.5 cursor-pointer hover:border-gray-300 transition-colors {selectedAmenities.includes(amenity.code) ? 'border-gray-900 bg-gray-50' : ''}">
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
      </div>
    </div>

    <div class="mt-8 flex justify-between">
      <button type="button" onclick={() => step = 1} class="text-sm text-gray-500 hover:text-gray-800">← Back</button>
      <button
        type="button"
        onclick={() => step = 3}
        class="rounded-xl bg-gray-900 px-6 py-2.5 text-sm font-semibold text-white hover:bg-gray-700 transition-colors"
      >
        Continue →
      </button>
    </div>

  <!-- Step 3: Pricing & rules -->
  {:else}
    <div class="space-y-5">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="price" class="block text-sm font-medium text-gray-700 mb-1">Price per night</label>
          <div class="flex">
            <select
              bind:value={currency}
              class="rounded-l-xl border border-r-0 border-gray-300 px-3 py-2.5 text-sm bg-gray-50 focus:outline-none"
            >
              {#each currencies as c}
                <option value={c}>{c}</option>
              {/each}
            </select>
            <input
              id="price"
              type="number"
              bind:value={pricePerNight}
              min="0"
              placeholder="0"
              class="flex-1 rounded-r-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
            />
          </div>
        </div>
        <div>
          <label for="cleaning" class="block text-sm font-medium text-gray-700 mb-1">Cleaning fee</label>
          <input
            id="cleaning"
            type="number"
            bind:value={cleaningFee}
            min="0"
            placeholder="0"
            class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="minNights" class="block text-sm font-medium text-gray-700 mb-1">Min nights</label>
          <input id="minNights" type="number" bind:value={minNights} min="1" class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none" />
        </div>
        <div>
          <label for="maxNights" class="block text-sm font-medium text-gray-700 mb-1">Max nights</label>
          <input id="maxNights" type="number" bind:value={maxNights} min="1" class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none" />
        </div>
      </div>

      <div>
        <p class="text-sm font-medium text-gray-700 mb-2">Cancellation policy</p>
        <div class="grid grid-cols-3 gap-2">
          {#each ['flexible', 'moderate', 'strict'] as policy}
            <button
              type="button"
              onclick={() => cancellationPolicy = policy}
              class="rounded-xl border px-3 py-2.5 text-sm font-medium capitalize transition-colors {cancellationPolicy === policy ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-200 text-gray-700 hover:border-gray-400'}"
            >
              {policy}
            </button>
          {/each}
        </div>
        <p class="mt-1.5 text-xs text-gray-400">
          {cancellationPolicy === 'flexible' ? 'Full refund 24+ hours before check-in.' : cancellationPolicy === 'moderate' ? 'Full refund 5+ days, 50% within 1–4 days.' : '50% refund 14+ days before check-in.'}
        </p>
      </div>

      <div>
        <p class="text-sm font-medium text-gray-700 mb-3">House rules</p>
        <div class="grid grid-cols-2 gap-4 mb-4">
          <div>
            <label for="checkInFrom" class="block text-xs text-gray-500 mb-1">Check-in from</label>
            <input id="checkInFrom" type="time" bind:value={checkInFrom} class="w-full rounded-xl border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
          </div>
          <div>
            <label for="checkOutBefore" class="block text-xs text-gray-500 mb-1">Check-out before</label>
            <input id="checkOutBefore" type="time" bind:value={checkOutBefore} class="w-full rounded-xl border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
          </div>
        </div>
        <div class="flex flex-col gap-2">
          {#each [
            { label: 'Smoking allowed', val: smoking, set: (v: boolean) => smoking = v },
            { label: 'Pets allowed',    val: pets,    set: (v: boolean) => pets = v },
            { label: 'Parties allowed', val: parties,  set: (v: boolean) => parties = v },
          ] as rule}
            <label class="flex items-center gap-3 cursor-pointer">
              <button
                type="button"
                role="switch"
                aria-checked={rule.val}
                onclick={() => rule.set(!rule.val)}
                class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors {rule.val ? 'bg-[#ff5a5f]' : 'bg-gray-200'}"
              >
                <span class="inline-block h-3.5 w-3.5 rounded-full bg-white shadow transition-transform {rule.val ? 'translate-x-4' : 'translate-x-1'}"></span>
              </button>
              <span class="text-sm text-gray-700">{rule.label}</span>
            </label>
          {/each}
        </div>
      </div>

      <div>
        <label class="flex items-center gap-3 cursor-pointer">
          <button
            type="button"
            role="switch"
            aria-checked={instantBook}
            onclick={() => instantBook = !instantBook}
            class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors {instantBook ? 'bg-[#ff5a5f]' : 'bg-gray-200'}"
          >
            <span class="inline-block h-3.5 w-3.5 rounded-full bg-white shadow transition-transform {instantBook ? 'translate-x-4' : 'translate-x-1'}"></span>
          </button>
          <span class="text-sm font-medium text-gray-700">Instant book</span>
        </label>
        <p class="ml-12 text-xs text-gray-400 mt-0.5">Guests can book without approval.</p>
      </div>
    </div>

    <div class="mt-8 flex justify-between">
      <button type="button" onclick={() => step = 2} class="text-sm text-gray-500 hover:text-gray-800">← Back</button>
      <button
        type="button"
        onclick={submit}
        disabled={submitting || !pricePerNight}
        class="rounded-xl bg-[#ff5a5f] px-6 py-2.5 text-sm font-semibold text-white hover:bg-[#e84f54] disabled:opacity-50 transition-colors"
      >
        {submitting ? 'Creating…' : 'Create listing'}
      </button>
    </div>
  {/if}
</div>
