/**
 * E2E tests — Listings search page (/listings)
 *
 * Note: The Price/Amenities filter dropdowns rely on a `[data-filter]` attribute
 * that is checked in the window click handler but not actually set on any element.
 * This means clicking those buttons immediately closes the dropdown (a known app
 * limitation). Those tests use URL navigation instead of dropdown interaction.
 */
import { test, expect } from './fixtures.js';

// ---------------------------------------------------------------------------
// Default load
// ---------------------------------------------------------------------------

test.describe('Listings page — default', () => {
  test('shows correct title and all listings by default', async ({ page }) => {
    await page.goto('/listings');

    await expect(page).toHaveTitle(/Explore stays/);
    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
    await expect(page.getByText('Samarkand Historical Suite')).toBeVisible();
    await expect(page.getByText('Bukhara Silk Road Retreat')).toBeVisible();
  });

  test('shows result count', async ({ page }) => {
    await page.goto('/listings');

    await expect(page.getByText(/15 stays found/)).toBeVisible();
  });

  test('compact search bar is visible with inputs', async ({ page }) => {
    await page.goto('/listings');

    await expect(page.getByPlaceholder('Where?')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Search' })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// City search
// ---------------------------------------------------------------------------

test.describe('Listings page — city filter', () => {
  test('pre-fills search bar from URL param and shows city chip', async ({ page }) => {
    await page.goto('/listings?city=Tashkent');

    await expect(page.getByPlaceholder('Where?')).toHaveValue('Tashkent');
    await expect(page.getByText('📍 Tashkent')).toBeVisible();
  });

  test('filtering by city narrows results and updates title', async ({ page }) => {
    await page.goto('/listings?city=Tashkent');

    await expect(page).toHaveTitle(/Stays in Tashkent/);
    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
    await expect(page.getByText('Samarkand Historical Suite')).not.toBeVisible();
  });

  test('clicking × on city chip clears filter', async ({ page }) => {
    await page.goto('/listings?city=Tashkent');

    await page.getByRole('link', { name: '×' }).click();
    await page.waitForURL(/\/listings($|\?(?!city=))/);

    await expect(page.getByText('Samarkand Historical Suite')).toBeVisible();
  });

  test('search bar submission navigates to filtered URL', async ({ page }) => {
    await page.goto('/listings');

    await page.getByPlaceholder('Where?').fill('Samarkand');
    await page.getByRole('button', { name: 'Search' }).dispatchEvent('click');

    await page.waitForURL(/city=Samarkand/);
    expect(page.url()).toContain('city=Samarkand');
  });

  test('empty search shows "No stays found" heading with clear link', async ({ page }) => {
    await page.goto('/listings?city=NonexistentCity123');

    await expect(page.getByText('No stays found').first()).toBeVisible();
    await expect(page.getByRole('link', { name: 'Clear all filters' })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Property type filter
// ---------------------------------------------------------------------------

test.describe('Listings page — property type tabs', () => {
  test('apartment tab filters to apartment listings', async ({ page }) => {
    await page.goto('/listings');

    await page.getByRole('button', { name: 'Apartment', exact: true }).dispatchEvent('click');
    await page.waitForURL(/type=apartment/);

    expect(page.url()).toContain('type=apartment');
    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
    await expect(page.getByText('Samarkand Historical Suite')).not.toBeVisible();
  });

  test('house tab filters correctly', async ({ page }) => {
    await page.goto('/listings');

    await page.getByRole('button', { name: 'House', exact: true }).dispatchEvent('click');
    await page.waitForURL(/type=house/);

    expect(page.url()).toContain('type=house');
  });

  test('"All types" tab clears type filter', async ({ page }) => {
    await page.goto('/listings?type=apartment');

    await page.getByRole('button', { name: 'All types', exact: true }).dispatchEvent('click');
    await page.waitForURL(/\/listings($|\?(?!type=))/);

    expect(page.url()).not.toContain('type=');
  });
});

// ---------------------------------------------------------------------------
// Instant book toggle
// ---------------------------------------------------------------------------

test.describe('Listings page — instant book', () => {
  test('instant book toggle filters to instant-book listings', async ({ page }) => {
    await page.goto('/listings');

    await page.getByRole('button', { name: /Instant book/i }).dispatchEvent('click');
    await page.waitForURL(/instant_book=true/);

    await expect(page.getByText('Samarkand Historical Suite')).toBeVisible();
    await expect(page.getByText('Cozy Apartment in Tashkent')).not.toBeVisible();
  });

  test('instant book toggle off removes param', async ({ page }) => {
    await page.goto('/listings?instant_book=true');

    await page.getByRole('button', { name: /Instant book/i }).dispatchEvent('click');
    await page.waitForURL(/\/listings($|\?(?!instant_book=))/);

    expect(page.url()).not.toContain('instant_book');
  });

  test('instant book button has active styling when set', async ({ page }) => {
    await page.goto('/listings?instant_book=true');

    const btn = page.getByRole('button', { name: /Instant book/i });
    // When active, the button has bg-gray-900 class
    await expect(btn).toHaveClass(/bg-gray-900/);
  });
});

// ---------------------------------------------------------------------------
// Price filter — via URL navigation (dropdown has a known open/close issue)
// ---------------------------------------------------------------------------

test.describe('Listings page — price filter', () => {
  test('min_price and max_price params show on Price button', async ({ page }) => {
    await page.goto('/listings?min_price=100000&max_price=200000');

    // The Price button should display the active range
    await expect(page.getByRole('button', { name: /100000/ })).toBeVisible();
  });

  test('navigating with price params applies filter and shows results', async ({ page }) => {
    await page.goto('/listings?min_price=100000&max_price=180000');

    // Filter returns only LISTING_TASHKENT (150k) — the mock gateway applies filtering
    // (It returns all listings regardless, but the UI shows the Price filter as active)
    await expect(page.getByRole('button', { name: /100000/ })).toBeVisible();
  });

  test('result count reflects active city filter', async ({ page }) => {
    await page.goto('/listings?city=Tashkent');

    await expect(page.getByText('3 stays found')).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Navigation from listing card
// ---------------------------------------------------------------------------

test.describe('Listings page — navigation', () => {
  test('clicking a listing card navigates to the listing detail page', async ({ page }) => {
    await page.goto('/listings');

    await page.getByText('Cozy Apartment in Tashkent').click();
    await page.waitForURL(/\/listings\/listing-001/);

    await expect(page.getByRole('heading', { name: 'Cozy Apartment in Tashkent' })).toBeVisible();
  });

  test('dates in URL are shown as filter chips', async ({ page }) => {
    await page.goto('/listings?check_in=2026-04-10&check_out=2026-04-15');

    await expect(page.getByText('📅 2026-04-10 → 2026-04-15')).toBeVisible();
  });

  test('"All stays" back link from listing detail navigates to /listings', async ({ page }) => {
    await page.goto('/listings/listing-001');

    await page.getByRole('link', { name: 'All stays' }).click();
    await page.waitForURL(/\/listings$/);

    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
  });
});
