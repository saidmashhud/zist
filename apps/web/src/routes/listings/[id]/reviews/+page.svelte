<script lang="ts">
  import type { PageData } from './$types';
  import type { Review } from './+page.ts';
  import { t } from '$lib/i18n';

  let { data }: { data: PageData } = $props();

  const listing  = $derived(data.listing);
  const reviews  = $derived(data.reviews as Review[]);
  const bookingId = $derived((data as any).bookingId as string);
  const hostId    = $derived((data as any).hostId as string);

  // Review form shown only when arriving from a completed booking
  const canReview = $derived(bookingId !== '');

  // Review form state
  let rating  = $state(0);
  let comment = $state('');
  let hover   = $state(0);
  let submitting = $state(false);
  let submitError = $state('');
  let submitted  = $state(false);

  async function submitReview() {
    if (rating < 1 || rating > 5) { submitError = 'Please select a rating.'; return; }
    if (!comment.trim()) { submitError = 'Please write a comment.'; return; }
    submitError = '';
    submitting = true;
    try {
      const res = await fetch('/api/reviews', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          bookingId,
          listingId: listing.id,
          hostId,
          rating,
          comment: comment.trim(),
        }),
      });
      if (!res.ok) {
        const e = await res.json().catch(() => ({}));
        submitError = e.error ?? 'Failed to submit review.';
      } else {
        submitted = true;
        rating = 0;
        comment = '';
      }
    } catch {
      submitError = 'Something went wrong. Please try again.';
    } finally {
      submitting = false;
    }
  }

  function starClass(n: number) {
    const active = hover > 0 ? n <= hover : n <= rating;
    return active ? 'text-yellow-400' : 'text-gray-300';
  }

  function formatDate(ts: number) {
    return new Date(ts * 1000).toLocaleDateString('en-GB', { month: 'short', year: 'numeric' });
  }
</script>

<svelte:head>
  <title>Reviews — {listing.title} — Zist</title>
</svelte:head>

<div class="mx-auto max-w-3xl px-6 py-8">

  <!-- Back -->
  <a
    href="/listings/{listing.id}"
    class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-900 transition-colors mb-6"
  >
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
    </svg>
    {listing.title}
  </a>

  <h1 class="text-2xl font-bold text-gray-900 mb-1">{$t.listing.reviews}</h1>
  {#if listing.averageRating > 0}
    <p class="text-gray-500 text-sm mb-8">
      {listing.averageRating.toFixed(1)} average · {listing.reviewCount} review{listing.reviewCount !== 1 ? 's' : ''}
    </p>
  {/if}

  <!-- Write a review (only shown when arriving from a completed booking) -->
  {#if canReview}
  <section class="mb-10 rounded-2xl border border-gray-200 p-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-4">{$t.reviews.write_review}</h2>

    {#if submitted}
      <div class="rounded-xl bg-green-50 border border-green-200 p-4 text-center">
        <p class="font-semibold text-green-800">Thank you for your review!</p>
        <p class="text-sm text-green-700 mt-1">Your review has been submitted.</p>
      </div>
    {:else}
      <!-- Star rating -->
      <div class="mb-4">
        <p class="text-sm font-medium text-gray-700 mb-2">{$t.reviews.your_rating}</p>
        <div class="flex gap-1">
          {#each [1, 2, 3, 4, 5] as n}
            <button
              type="button"
              onmouseenter={() => hover = n}
              onmouseleave={() => hover = 0}
              onclick={() => rating = n}
              class="text-3xl transition-colors {starClass(n)}"
              aria-label="Rate {n} stars"
            >★</button>
          {/each}
        </div>
      </div>

      <!-- Comment -->
      <div class="mb-4">
        <label for="review-comment" class="block text-sm font-medium text-gray-700 mb-2">
          {$t.reviews.your_comment}
        </label>
        <textarea
          id="review-comment"
          bind:value={comment}
          rows="4"
          placeholder="Share your experience with other travellers…"
          class="w-full rounded-xl border border-gray-300 px-4 py-3 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-[#ff5a5f]/30 focus:border-[#ff5a5f] resize-none"
        ></textarea>
      </div>

      {#if submitError}
        <p class="mb-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{submitError}</p>
      {/if}

      <button
        type="button"
        onclick={submitReview}
        disabled={submitting}
        class="rounded-xl bg-[#ff5a5f] px-6 py-2.5 text-sm font-semibold text-white hover:bg-[#e84f54] transition-colors disabled:opacity-50"
      >
        {submitting ? 'Submitting…' : $t.reviews.submit}
      </button>
    {/if}
  </section>
  {/if}

  <!-- Reviews list -->
  {#if reviews.length === 0}
    <p class="text-gray-500 text-center py-8">{$t.listing.no_reviews}</p>
  {:else}
    <div class="space-y-6">
      {#each reviews as review (review.id)}
        <div class="border-b border-gray-100 pb-6 last:border-0">
          <!-- Rating + date -->
          <div class="flex items-center justify-between mb-2">
            <div class="flex gap-0.5">
              {#each [1, 2, 3, 4, 5] as n}
                <span class="text-sm {n <= review.rating ? 'text-yellow-400' : 'text-gray-200'}">★</span>
              {/each}
            </div>
            <span class="text-xs text-gray-400">{formatDate(review.createdAt)}</span>
          </div>

          <!-- Comment -->
          <p class="text-gray-700 text-sm leading-relaxed">{review.comment}</p>

          <!-- Host reply -->
          {#if review.reply}
            <div class="mt-3 ml-4 pl-4 border-l-2 border-gray-200">
              <p class="text-xs font-semibold text-gray-500 mb-1">{$t.reviews.host_reply}</p>
              <p class="text-sm text-gray-600">{review.reply}</p>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}

</div>
