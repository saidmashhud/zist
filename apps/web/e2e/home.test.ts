/**
 * E2E tests — Home page (/)
 */
import { test, expect } from './fixtures.js';

// ---------------------------------------------------------------------------
// Unauthenticated home page
// ---------------------------------------------------------------------------

test.describe('Home page — unauthenticated', () => {
  test('renders hero section with headline and search form', async ({ page }) => {
    await page.goto('/');

    await expect(page).toHaveTitle(/Zist — Find your stay in Central Asia/);
    await expect(page.getByRole('heading', { name: /Find your stay/i })).toBeVisible();
    await expect(page.getByPlaceholder('City or destination')).toBeVisible();
    await expect(page.getByLabel('Check-in')).toBeVisible();
    await expect(page.getByLabel('Check-out')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Search' })).toBeVisible();
  });

  test('navbar shows logo, explore, my trips, and sign-in', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('link', { name: 'zist' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Explore' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'My trips' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Host dashboard' })).not.toBeVisible();
  });

  test('popular destinations section renders first 6 cities as links', async ({ page }) => {
    await page.goto('/');

    const cities = ['Tashkent', 'Samarkand', 'Bukhara', 'Almaty', 'Astana', 'Bishkek'];
    for (const city of cities) {
      await expect(page.getByRole('link', { name: city }).first()).toBeVisible();
    }
  });

  test('featured listings grid shows cards from API', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
    await expect(page.getByText('Samarkand Historical Suite')).toBeVisible();
  });

  test('search form submits to /listings with correct query params', async ({ page }) => {
    await page.goto('/');

    await page.getByPlaceholder('City or destination').fill('Tashkent');
    await page.getByLabel('Check-in').fill('2026-04-10');
    await page.getByLabel('Check-out').fill('2026-04-15');

    // Increment guests to 3 via + button (inside the Guests block)
    const plusBtn = page.getByRole('button', { name: '+' }).first();
    await plusBtn.dispatchEvent('click');
    await plusBtn.dispatchEvent('click');

    await page.getByRole('button', { name: 'Search' }).dispatchEvent('click');
    await page.waitForURL(/\/listings/);

    const url = page.url();
    expect(url).toContain('/listings');
    expect(url).toContain('city=Tashkent');
    expect(url).toContain('check_in=2026-04-10');
    expect(url).toContain('check_out=2026-04-15');
    expect(url).toContain('guests=3');
  });

  test('clicking a popular city link navigates to /listings?city=...', async ({ page }) => {
    await page.goto('/');

    await page.getByRole('link', { name: 'Tashkent' }).first().click();
    await page.waitForURL(/city=Tashkent/);

    expect(page.url()).toContain('city=Tashkent');
  });

  test('logo link navigates back to home', async ({ page }) => {
    await page.goto('/listings');
    await page.getByRole('link', { name: 'zist' }).click();
    await page.waitForURL('/');
    await expect(page.getByRole('heading', { name: /Find your stay/i })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Authenticated home page
// ---------------------------------------------------------------------------

test.describe('Home page — authenticated', () => {
  test('shows user email and sign-out button in navbar', async ({ authedPage: page }) => {
    await page.goto('/');

    await expect(page.getByText('test@zist.test')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Sign out' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Sign in' })).not.toBeVisible();
  });

  test('shows Host dashboard link when authenticated', async ({ authedPage: page }) => {
    await page.goto('/');

    await expect(page.getByRole('link', { name: 'Host dashboard' })).toBeVisible();
  });

  test('sign-out button calls logout endpoint', async ({ authedPage: page }) => {
    await page.goto('/');

    let logoutCalled = false;
    await page.route('**/api/auth/logout', route => {
      logoutCalled = true;
      return route.fulfill({ status: 200 });
    });

    await page.getByRole('button', { name: 'Sign out' }).dispatchEvent('click');
    await page.waitForTimeout(500);
    expect(logoutCalled).toBe(true);
  });

  test('footer is visible on home page', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText(/stays across Central Asia/i)).toBeVisible();
  });
});
