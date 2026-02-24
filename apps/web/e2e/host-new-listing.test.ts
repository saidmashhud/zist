/**
 * E2E tests — Create new listing (/host/listings/new)
 */
import { test, expect } from './fixtures.js';
import { LISTING_TASHKENT } from './mock-data.js';

// ---------------------------------------------------------------------------
// Auth guard
// ---------------------------------------------------------------------------

test.describe('New listing — auth guard', () => {
  test('unauthenticated access redirects to /login', async ({ page }) => {
    await page.goto('/host/listings/new');
    await expect(page).toHaveURL(/\/login/);
  });
});

// ---------------------------------------------------------------------------
// Multi-step form
// ---------------------------------------------------------------------------

test.describe('New listing — multi-step form', () => {
  test('shows "Create a new listing" heading on step 1', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    await expect(page.getByRole('heading', { name: 'Create a new listing' })).toBeVisible();
    await expect(page.getByText('Location & type')).toBeVisible();
  });

  test('Continue button is disabled when required fields are empty', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    const continueBtn = page.getByRole('button', { name: 'Continue →' });
    await expect(continueBtn).toBeDisabled();
  });

  test('filling required fields enables the Continue button', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    await page.fill('#title', 'Test Listing');
    await page.fill('#city', 'Tashkent');
    await page.fill('#country', 'Uzbekistan');

    await expect(page.getByRole('button', { name: 'Continue →' })).not.toBeDisabled();
  });

  test('Continue advances to step 2', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    await page.fill('#title', 'Test Listing');
    await page.fill('#city', 'Tashkent');
    await page.fill('#country', 'Uzbekistan');

    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    await expect(page.getByText('Details & amenities')).toBeVisible();
  });

  test('Back button on step 2 returns to step 1', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    await page.fill('#title', 'Test Listing');
    await page.fill('#city', 'Tashkent');
    await page.fill('#country', 'Uzbekistan');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    await expect(page.getByText('Details & amenities')).toBeVisible();

    await page.getByRole('button', { name: '← Back' }).dispatchEvent('click');

    await expect(page.getByText('Location & type')).toBeVisible();
  });

  test('Continue on step 2 advances to step 3', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    // Step 1
    await page.fill('#title', 'Test Listing');
    await page.fill('#city', 'Tashkent');
    await page.fill('#country', 'Uzbekistan');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    // Step 2
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    await expect(page.getByText('Pricing & rules')).toBeVisible();
  });

  test('Create listing button is disabled when price is empty on step 3', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    // Navigate to step 3
    await page.fill('#title', 'Test Listing');
    await page.fill('#city', 'Tashkent');
    await page.fill('#country', 'Uzbekistan');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    const createBtn = page.getByRole('button', { name: 'Create listing' });
    await expect(createBtn).toBeDisabled();
  });

  test('Back button on step 3 returns to step 2', async ({ hostPage: page }) => {
    await page.goto('/host/listings/new');

    await page.fill('#title', 'Test Listing');
    await page.fill('#city', 'Tashkent');
    await page.fill('#country', 'Uzbekistan');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    await expect(page.getByText('Pricing & rules')).toBeVisible();
    await page.getByRole('button', { name: '← Back' }).dispatchEvent('click');

    await expect(page.getByText('Details & amenities')).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Submission
// ---------------------------------------------------------------------------

test.describe('New listing — submission', () => {
  test('submitting all steps POSTs to /api/listings and redirects to edit', async ({ hostPage: page }) => {
    const newListing = { ...LISTING_TASHKENT, id: 'listing-new-001', title: 'My New Listing' };

    // Intercept POST /api/listings
    await page.route('**/api/listings', async route => {
      if (route.request().method() !== 'POST') { await route.continue(); return; }
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(newListing),
      });
    });

    // Intercept subsequent GET for the new listing (page.ts load function)
    await page.route('**/api/listings/listing-new-001', async route => {
      if (route.request().method() !== 'GET') { await route.continue(); return; }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(newListing),
      });
    });

    await page.goto('/host/listings/new');

    // Step 1
    await page.fill('#title', 'My New Listing');
    await page.fill('#city', 'Tashkent');
    await page.fill('#country', 'Uzbekistan');
    await page.fill('#address', 'Test St 1');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    // Step 2
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');

    // Step 3 — fill price
    await page.fill('#price', '150000');

    const postReq = page.waitForRequest(
      req => new URL(req.url()).pathname === '/api/listings' && req.method() === 'POST'
    );
    await page.getByRole('button', { name: 'Create listing' }).dispatchEvent('click');
    await postReq;

    await page.waitForURL(/\/host\/listings\/listing-new-001\/edit\?created=1/);
  });

  test('POST body contains title, city, country from form', async ({ hostPage: page }) => {
    const newListing = { ...LISTING_TASHKENT, id: 'listing-new-001', title: 'My Test Listing' };

    await page.route('**/api/listings', async route => {
      if (route.request().method() !== 'POST') { await route.continue(); return; }
      await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(newListing) });
    });

    await page.route('**/api/listings/listing-new-001', async route => {
      if (route.request().method() !== 'GET') { await route.continue(); return; }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(newListing) });
    });

    await page.goto('/host/listings/new');

    await page.fill('#title', 'My Test Listing');
    await page.fill('#city', 'Bishkek');
    await page.fill('#country', 'Kyrgyzstan');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');
    await page.getByRole('button', { name: 'Continue →' }).dispatchEvent('click');
    await page.fill('#price', '5000');

    const postReq = page.waitForRequest(
      req => new URL(req.url()).pathname === '/api/listings' && req.method() === 'POST'
    );
    await page.getByRole('button', { name: 'Create listing' }).dispatchEvent('click');
    const req = await postReq;

    // Use the request object directly to get the body (avoids race condition with route callback)
    const body = JSON.parse(req.postData() ?? '{}');
    expect(body).toMatchObject({ title: 'My Test Listing', city: 'Bishkek', country: 'Kyrgyzstan' });
  });
});
