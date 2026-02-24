/**
 * E2E tests — Listing detail page (/listings/[id])
 */
import { test, expect } from './fixtures.js';
import { PRICE_PREVIEW_3N, CHECKOUT_SESSION } from './mock-data.js';

// ---------------------------------------------------------------------------
// Listing detail — general content
// ---------------------------------------------------------------------------

test.describe('Listing detail — content', () => {
  test('renders title, city, and property type span', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page).toHaveTitle(/Cozy Apartment in Tashkent/);
    await expect(page.getByRole('heading', { name: 'Cozy Apartment in Tashkent' })).toBeVisible();
    await expect(page.getByText('Tashkent, Uzbekistan')).toBeVisible();
    // The type is rendered in a <span class="text-sm">
    await expect(page.locator('span.text-sm').filter({ hasText: 'Apartment' })).toBeVisible();
  });

  test('renders property stats: bedrooms, beds, bathrooms, guests', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByText('2 bedrooms')).toBeVisible();
    await expect(page.getByText('2 beds')).toBeVisible();
    await expect(page.getByText('1 bath')).toBeVisible();
    await expect(page.getByText('Up to 4 guests')).toBeVisible();
  });

  test('renders description', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByText('A charming apartment in the heart of Tashkent')).toBeVisible();
  });

  test('renders amenities section', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByText('What this place offers')).toBeVisible();
    await expect(page.getByText('Wi-Fi')).toBeVisible();
    await expect(page.getByText('Kitchen')).toBeVisible();
  });

  test('renders house rules when present', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByText('House rules')).toBeVisible();
    await expect(page.getByText('Check-in after')).toBeVisible();
    await expect(page.getByText('14:00')).toBeVisible();
    await expect(page.getByText('No smoking')).toBeVisible();
    await expect(page.getByText('Pets allowed')).toBeVisible();
    await expect(page.getByText('No parties or events')).toBeVisible();
  });

  test('renders cancellation policy', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByText('Cancellation policy')).toBeVisible();
    await expect(page.getByText('Flexible')).toBeVisible();
    await expect(page.getByText(/Full refund if cancelled at least 24 hours/)).toBeVisible();
  });

  test('shows instant book badge for instant-book listings', async ({ page }) => {
    await page.goto('/listings/listing-002');

    await expect(page.getByText('Instant book').first()).toBeVisible();
  });

  test('shows star rating when averageRating > 0', async ({ page }) => {
    await page.goto('/listings/listing-001');

    // Rating appears in two places (title row + booking widget), use first()
    await expect(page.getByText('4.8').first()).toBeVisible();
    await expect(page.getByText('12 reviews')).toBeVisible();
  });

  test('no star rating section when reviewCount is 0', async ({ page }) => {
    await page.goto('/listings/listing-002');

    // averageRating=0 → the rating span is not rendered at all
    await expect(page.getByText(/reviews/)).not.toBeVisible();
  });

  test('"All stays" back link is visible', async ({ page }) => {
    await page.goto('/listings/listing-001');
    await expect(page.getByRole('link', { name: 'All stays' })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Booking widget — unauthenticated
// ---------------------------------------------------------------------------

test.describe('Booking widget — unauthenticated', () => {
  test('shows "Sign in to book" prompt instead of date pickers', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByText('Sign in to book this stay')).toBeVisible();
    await expect(page.getByRole('link', { name: 'Sign in to book' })).toBeVisible();
    await expect(page.getByLabel('Check-in')).not.toBeVisible();
  });

  test('price per night is always visible', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByText('150000')).toBeVisible();
    await expect(page.getByText('UZS / night')).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Booking widget — authenticated (request-approval flow)
// ---------------------------------------------------------------------------

test.describe('Booking widget — authenticated, non-instant', () => {
  test('shows date pickers and guests selector', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-001');

    await expect(page.getByLabel('Check-in')).toBeVisible();
    await expect(page.getByLabel('Check-out')).toBeVisible();
    await expect(page.getByLabel('Guests')).toBeVisible();
  });

  test('"Request to book" button is disabled until dates are selected', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-001');

    const btn = page.getByRole('button', { name: 'Request to book' });
    await expect(btn).toBeVisible();
    await expect(btn).toBeDisabled();
  });

  test('selecting dates triggers price preview and shows breakdown', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-001');

    await page.route('**/api/listings/listing-001/price-preview**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(PRICE_PREVIEW_3N) })
    );

    await page.getByLabel('Check-in').fill('2026-04-10');
    await page.getByLabel('Check-out').fill('2026-04-13');

    await expect(page.getByText(/150000 × 3 night/)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('30000')).toBeVisible();   // cleaning fee
    await expect(page.getByText('33000')).toBeVisible();   // service fee
    await expect(page.getByText('513000').first()).toBeVisible();  // total (also in price line)
  });

  test('"Request to book" button enables after valid dates + price loaded', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-001');

    await page.route('**/api/listings/listing-001/price-preview**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(PRICE_PREVIEW_3N) })
    );

    await page.getByLabel('Check-in').fill('2026-04-10');
    await page.getByLabel('Check-out').fill('2026-04-13');
    await page.waitForResponse('**/api/listings/listing-001/price-preview**');

    await expect(page.getByRole('button', { name: 'Request to book' })).toBeEnabled({ timeout: 5000 });
  });

  test('submitting booking shows "Request sent!" success banner', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-001');

    await page.route('**/api/listings/listing-001/price-preview**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(PRICE_PREVIEW_3N) })
    );
    await page.route('**/api/bookings', async route => {
      if (route.request().method() !== 'POST') { await route.continue(); return; }
      await route.fulfill({
        status: 201, contentType: 'application/json',
        body: JSON.stringify({ id: 'booking-new-001', listingId: 'listing-001', status: 'pending_host_approval' }),
      });
    });

    await page.getByLabel('Check-in').fill('2026-04-10');
    await page.getByLabel('Check-out').fill('2026-04-13');
    await page.waitForResponse('**/api/listings/listing-001/price-preview**');
    await expect(page.getByRole('button', { name: 'Request to book' })).toBeEnabled({ timeout: 5000 });
    await page.getByRole('button', { name: 'Request to book' }).dispatchEvent('click');

    await expect(page.getByText('Request sent!')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('The host will review your request')).toBeVisible();
    await expect(page.getByRole('link', { name: 'View my bookings →' })).toBeVisible();
  });

  test('409 conflict shows dates-unavailable error', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-001');

    await page.route('**/api/listings/listing-001/price-preview**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(PRICE_PREVIEW_3N) })
    );
    await page.route('**/api/bookings', async route => {
      if (route.request().method() !== 'POST') { await route.continue(); return; }
      await route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ error: 'conflict' }) });
    });

    await page.getByLabel('Check-in').fill('2026-04-10');
    await page.getByLabel('Check-out').fill('2026-04-13');
    await page.waitForResponse('**/api/listings/listing-001/price-preview**');
    await page.getByRole('button', { name: 'Request to book' }).dispatchEvent('click');

    await expect(page.getByText('Selected dates are no longer available')).toBeVisible({ timeout: 5000 });
  });

  test('price preview error is shown when API returns error', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-001');

    await page.route('**/api/listings/listing-001/price-preview**', route =>
      route.fulfill({ status: 400, contentType: 'application/json', body: JSON.stringify({ error: 'minimum stay not met' }) })
    );

    await page.getByLabel('Check-in').fill('2026-04-10');
    await page.getByLabel('Check-out').fill('2026-04-11');
    await page.waitForResponse('**/api/listings/listing-001/price-preview**');

    await expect(page.getByText('minimum stay not met')).toBeVisible({ timeout: 5000 });
  });
});

// ---------------------------------------------------------------------------
// Booking widget — instant book
// ---------------------------------------------------------------------------

test.describe('Booking widget — instant book listing', () => {
  test('shows "Reserve" button for instant-book listings', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-002');

    await expect(page.getByRole('button', { name: 'Reserve' })).toBeVisible();
  });

  test('shows "You won\'t be charged yet" caption', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-002');

    await expect(page.getByText(/You won't be charged yet/)).toBeVisible();
  });

  test('instant book calls checkout API with correct payload', async ({ authedPage: page }) => {
    await page.goto('/listings/listing-002');

    await page.route('**/api/listings/listing-002/price-preview**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(PRICE_PREVIEW_3N) })
    );
    await page.route('**/api/bookings', async route => {
      if (route.request().method() !== 'POST') { await route.continue(); return; }
      await route.fulfill({
        status: 201, contentType: 'application/json',
        body: JSON.stringify({ id: 'booking-new-002', listingId: 'listing-002', status: 'payment_pending' }),
      });
    });
    await page.route('**/api/payments/checkout', route =>
      route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(CHECKOUT_SESSION) })
    );

    await page.getByLabel('Check-in').fill('2026-04-10');
    await page.getByLabel('Check-out').fill('2026-04-13');
    await page.waitForResponse('**/api/listings/listing-002/price-preview**');
    await page.getByRole('button', { name: 'Reserve' }).dispatchEvent('click');

    const checkoutReq = await page.waitForRequest('**/api/payments/checkout');
    expect(checkoutReq.method()).toBe('POST');
    const body = checkoutReq.postDataJSON();
    expect(body).toMatchObject({ listingId: 'listing-002', bookingId: 'booking-new-002' });
  });
});
