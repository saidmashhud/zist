<script lang="ts">
  import { goto } from '$app/navigation';
  import { AMENITIES } from '$lib/types';
  import type { PageData, } from './$types';
  import type { PropertyType, CancellationPolicy } from '$lib/types';

  let { data }: { data: PageData } = $props();

  const listing = $derived(data.listing);

  // Form state initialised from listing
  let title       = $state(listing.title);
  let description = $state(listing.description);
  let city        = $state(listing.city);
  let country     = $state(listing.country);
  let address     = $state(listing.address);
  let type        = $state<PropertyType>(listing.type);
  let bedrooms    = $state(listing.bedrooms);
  let beds        = $state(listing.beds);
  let bathrooms   = $state(listing.bathrooms);
  let maxGuests   = $state(listing.maxGuests);
  let pricePerNight    = $state(listing.pricePerNight);
  let currency         = $state(listing.currency);
  let cleaningFee      = $state(listing.cleaningFee);
  let minNights        = $state(listing.minNights);
  let maxNights        = $state(listing.maxNights);
  let cancellationPolicy = $state<CancellationPolicy>(listing.cancellationPolicy);
  let instantBook      = $state(listing.instantBook);
  let selectedAmenities = $state<string[]>([...listing.amenities]);

  let checkInFrom    = $state(listing.rules?.checkInFrom ?? '14:00');
  let checkOutBefore = $state(listing.rules?.checkOutBefore ?? '12:00');
  let smoking = $state(listing.rules?.smoking ?? false);
  let pets    = $state(listing.rules?.pets ?? false);
  let parties = $state(listing.rules?.parties ?? false);

  // Photo upload
  let photoUrl = $state('');
  let photoCaption = $state('');
  let uploadingPhoto = $state(false);
  let photoError = $state('');

  let saving  = $state(false);
  let saveError = $state('');
  let saved = $state(false);
  let publishing = $state(false);

  function toggleAmenity(code: string) {
    if (selectedAmenities.includes(code)) {
      selectedAmenities = selectedAmenities.filter(a => a !== code);
    } else {
      selectedAmenities = [...selectedAmenities, code];
    }
  }

  async function save() {
    saving = true;
    saveError = '';
    saved = false;
    try {
      const res = await fetch(`/api/listings/${listing.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title, description, city, country, address, type,
          bedrooms, beds, bathrooms, maxGuests,
          pricePerNight, currency, cleaningFee,
          minNights, maxNights,
          cancellationPolicy, instantBook,
          amenities: selectedAmenities,
          rules: { checkInFrom, checkOutBefore, smoking, pets, parties },
        }),
      });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        saveError = d.error ?? 'Failed to save.';
      } else {
        saved = true;
        setTimeout(() => saved = false, 3000);
      }
    } catch {
      saveError = 'Network error.';
    } finally {
      saving = false;
    }
  }

  async function publish() {
    publishing = true;
    const route = listing.status === 'active' ? 'unpublish' : 'publish';
    await fetch(`/api/listings/${listing.id}/${route}`, { method: 'POST' });
    publishing = false;
    goto(`/host`);
  }

  async function addPhoto() {
    if (!photoUrl) return;
    uploadingPhoto = true;
    photoError = '';
    try {
      const res = await fetch(`/api/listings/${listing.id}/photos`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: photoUrl, caption: photoCaption }),
      });
      if (!res.ok) {
        photoError = 'Failed to add photo.';
      } else {
        photoUrl = '';
        photoCaption = '';
        location.reload();
      }
    } catch {
      photoError = 'Network error.';
    } finally {
      uploadingPhoto = false;
    }
  }

  async function deletePhoto(photoId: string) {
    if (!confirm('Remove this photo?')) return;
    await fetch(`/api/listings/${listing.id}/photos/${photoId}`, { method: 'DELETE' });
    location.reload();
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
  <title>Edit listing — Zist</title>
</svelte:head>

<div class="mx-auto max-w-2xl px-6 py-10">

  <div class="flex items-center justify-between mb-6">
    <a href="/host" class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
      </svg>
      Dashboard
    </a>
    <div class="flex gap-2">
      <a
        href="/listings/{listing.id}"
        target="_blank"
        class="rounded-xl border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:border-gray-400 transition-colors"
      >
        Preview
      </a>
      <button
        onclick={publish}
        disabled={publishing}
        class="rounded-xl border px-4 py-2 text-sm font-medium transition-colors {listing.status === 'active' ? 'border-yellow-300 text-yellow-700 hover:bg-yellow-50' : 'border-green-300 text-green-700 hover:bg-green-50'}"
      >
        {publishing ? '…' : listing.status === 'active' ? 'Pause listing' : 'Publish listing'}
      </button>
    </div>
  </div>

  {#if data.created}
    <div class="mb-6 rounded-xl bg-green-50 border border-green-200 px-4 py-3 text-sm text-green-700">
      Listing created! Now add photos and publish when ready.
    </div>
  {/if}

  <h1 class="text-2xl font-bold text-gray-900 mb-6">{listing.title}</h1>

  <!-- Photos section -->
  <div class="mb-8">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Photos</h2>
    {#if listing.photos && listing.photos.length > 0}
      <div class="grid grid-cols-3 gap-2 mb-3">
        {#each listing.photos as photo (photo.id)}
          <div class="relative group">
            <img src={photo.url} alt={photo.caption || listing.title} class="w-full h-24 object-cover rounded-xl" />
            <button
              onclick={() => deletePhoto(photo.id)}
              class="absolute top-1 right-1 hidden group-hover:flex w-6 h-6 rounded-full bg-black/60 text-white items-center justify-center text-xs"
            >×</button>
          </div>
        {/each}
      </div>
    {:else}
      <p class="text-sm text-gray-400 mb-3">No photos yet.</p>
    {/if}
    <div class="flex gap-2">
      <input
        type="url"
        bind:value={photoUrl}
        placeholder="Photo URL"
        class="flex-1 rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none"
      />
      <input
        type="text"
        bind:value={photoCaption}
        placeholder="Caption (optional)"
        class="w-36 rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none"
      />
      <button
        onclick={addPhoto}
        disabled={uploadingPhoto || !photoUrl}
        class="rounded-xl bg-gray-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-40 hover:bg-gray-700 transition-colors"
      >
        {uploadingPhoto ? '…' : 'Add'}
      </button>
    </div>
    {#if photoError}<p class="mt-1 text-xs text-red-500">{photoError}</p>{/if}
  </div>

  <!-- Main form -->
  <div class="space-y-5">
    <div>
      <label for="edit-title" class="block text-sm font-medium text-gray-700 mb-1">Title</label>
      <input id="edit-title" type="text" bind:value={title} class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none" />
    </div>

    <div>
      <label for="edit-desc" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
      <textarea id="edit-desc" bind:value={description} rows="4" class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none resize-none"></textarea>
    </div>

    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="edit-city" class="block text-sm font-medium text-gray-700 mb-1">City</label>
        <input id="edit-city" type="text" bind:value={city} class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none" />
      </div>
      <div>
        <label for="edit-country" class="block text-sm font-medium text-gray-700 mb-1">Country</label>
        <input id="edit-country" type="text" bind:value={country} class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none" />
      </div>
    </div>

    <div>
      <label for="edit-address" class="block text-sm font-medium text-gray-700 mb-1">Address</label>
      <input id="edit-address" type="text" bind:value={address} class="w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm focus:border-[#ff5a5f] focus:outline-none" />
    </div>

    <div>
      <p class="text-sm font-medium text-gray-700 mb-2">Property type</p>
      <div class="flex gap-2 flex-wrap">
        {#each propertyTypes as pt}
          <button type="button" onclick={() => type = pt.value as PropertyType} class="rounded-xl border px-3 py-2 text-sm font-medium transition-colors {type === pt.value ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-200 text-gray-700 hover:border-gray-400'}">
            {pt.label}
          </button>
        {/each}
      </div>
    </div>

    <div class="grid grid-cols-4 gap-3">
      {#each [
        { label: 'Bedrooms', val: bedrooms, set: (v: number) => bedrooms = v },
        { label: 'Beds',     val: beds,     set: (v: number) => beds = v },
        { label: 'Baths',    val: bathrooms, set: (v: number) => bathrooms = v },
        { label: 'Guests',   val: maxGuests, set: (v: number) => maxGuests = v },
      ] as f}
        <div class="rounded-xl border border-gray-200 p-3 text-center">
          <p class="text-xs text-gray-500 mb-2">{f.label}</p>
          <div class="flex items-center justify-center gap-2">
            <button type="button" onclick={() => f.set(Math.max(1, f.val - 1))} class="w-5 h-5 rounded-full border border-gray-300 text-xs text-gray-600 leading-none">−</button>
            <span class="text-sm font-semibold w-4 text-center">{f.val}</span>
            <button type="button" onclick={() => f.set(Math.min(20, f.val + 1))} class="w-5 h-5 rounded-full border border-gray-300 text-xs text-gray-600 leading-none">+</button>
          </div>
        </div>
      {/each}
    </div>

    <div>
      <p class="text-sm font-medium text-gray-700 mb-2">Amenities</p>
      <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
        {#each AMENITIES as amenity}
          <label class="flex items-center gap-2 rounded-xl border border-gray-200 px-3 py-2 cursor-pointer hover:border-gray-300 {selectedAmenities.includes(amenity.code) ? 'border-gray-900 bg-gray-50' : ''}">
            <input type="checkbox" checked={selectedAmenities.includes(amenity.code)} onchange={() => toggleAmenity(amenity.code)} class="rounded border-gray-300 text-[#ff5a5f] focus:ring-[#ff5a5f]" />
            <span class="text-sm text-gray-700">{amenity.label}</span>
          </label>
        {/each}
      </div>
    </div>

    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="edit-price" class="block text-sm font-medium text-gray-700 mb-1">Price / night</label>
        <div class="flex">
          <select bind:value={currency} class="rounded-l-xl border border-r-0 border-gray-300 px-2 py-2 text-sm bg-gray-50 focus:outline-none">
            {#each currencies as c}<option value={c}>{c}</option>{/each}
          </select>
          <input id="edit-price" type="number" bind:value={pricePerNight} min="0" class="flex-1 rounded-r-xl border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
        </div>
      </div>
      <div>
        <label for="edit-cleaning" class="block text-sm font-medium text-gray-700 mb-1">Cleaning fee</label>
        <input id="edit-cleaning" type="number" bind:value={cleaningFee} min="0" class="w-full rounded-xl border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
      </div>
    </div>

    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="edit-min" class="block text-sm font-medium text-gray-700 mb-1">Min nights</label>
        <input id="edit-min" type="number" bind:value={minNights} min="1" class="w-full rounded-xl border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
      </div>
      <div>
        <label for="edit-max" class="block text-sm font-medium text-gray-700 mb-1">Max nights</label>
        <input id="edit-max" type="number" bind:value={maxNights} min="1" class="w-full rounded-xl border border-gray-300 px-4 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
      </div>
    </div>

    <div>
      <p class="text-sm font-medium text-gray-700 mb-2">Cancellation policy</p>
      <div class="flex gap-2">
        {#each (['flexible', 'moderate', 'strict'] as CancellationPolicy[]) as policy}
          <button type="button" onclick={() => cancellationPolicy = policy} class="rounded-xl border px-4 py-2 text-sm font-medium capitalize transition-colors {cancellationPolicy === policy ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-200 text-gray-700 hover:border-gray-400'}">
            {policy}
          </button>
        {/each}
      </div>
    </div>

    <div>
      <p class="text-sm font-medium text-gray-700 mb-3">House rules</p>
      <div class="grid grid-cols-2 gap-3 mb-3">
        <div>
          <label for="edit-checkin" class="block text-xs text-gray-500 mb-1">Check-in from</label>
          <input id="edit-checkin" type="time" bind:value={checkInFrom} class="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
        </div>
        <div>
          <label for="edit-checkout" class="block text-xs text-gray-500 mb-1">Check-out before</label>
          <input id="edit-checkout" type="time" bind:value={checkOutBefore} class="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-[#ff5a5f] focus:outline-none" />
        </div>
      </div>
      <div class="flex flex-col gap-2">
        {#each [
          { label: 'Smoking allowed', val: smoking, set: (v: boolean) => smoking = v },
          { label: 'Pets allowed',    val: pets,    set: (v: boolean) => pets = v },
          { label: 'Parties allowed', val: parties,  set: (v: boolean) => parties = v },
        ] as rule}
          <label class="flex items-center gap-3 cursor-pointer">
            <button type="button" role="switch" aria-checked={rule.val} onclick={() => rule.set(!rule.val)}
              class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors {rule.val ? 'bg-[#ff5a5f]' : 'bg-gray-200'}">
              <span class="inline-block h-3.5 w-3.5 rounded-full bg-white shadow transition-transform {rule.val ? 'translate-x-4' : 'translate-x-1'}"></span>
            </button>
            <span class="text-sm text-gray-700">{rule.label}</span>
          </label>
        {/each}
      </div>
    </div>

    <div>
      <label class="flex items-center gap-3 cursor-pointer">
        <button type="button" role="switch" aria-checked={instantBook} onclick={() => instantBook = !instantBook}
          class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors {instantBook ? 'bg-[#ff5a5f]' : 'bg-gray-200'}">
          <span class="inline-block h-3.5 w-3.5 rounded-full bg-white shadow transition-transform {instantBook ? 'translate-x-4' : 'translate-x-1'}"></span>
        </button>
        <span class="text-sm font-medium text-gray-700">Instant book</span>
      </label>
    </div>

    {#if saveError}
      <p class="text-sm text-red-500">{saveError}</p>
    {/if}

    {#if saved}
      <div class="rounded-xl bg-green-50 border border-green-200 px-4 py-2 text-sm text-green-700">
        Changes saved.
      </div>
    {/if}

    <div class="flex justify-end pt-2">
      <button
        onclick={save}
        disabled={saving}
        class="rounded-xl bg-[#ff5a5f] px-6 py-2.5 text-sm font-semibold text-white hover:bg-[#e84f54] disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Save changes'}
      </button>
    </div>
  </div>

</div>
