/**
 * E2E tests — Login page (/login)
 */
import { test, expect } from './fixtures.js';

test.describe('Login page', () => {
  test('shows sign-in heading and branding', async ({ page }) => {
    await page.goto('/login');

    await expect(page.getByText('Sign in to your account')).toBeVisible();
    // zist brand link in the login form (scoped to main, not the nav)
    await expect(page.getByRole('main').getByRole('link', { name: 'zist' })).toBeVisible();
  });

  test('has email and password inputs', async ({ page }) => {
    await page.goto('/login');

    await expect(page.locator('input#email')).toBeVisible();
    await expect(page.locator('input#password')).toBeVisible();
  });

  test('has a Sign in submit button', async ({ page }) => {
    await page.goto('/login');

    await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
  });

  test('shows error on invalid credentials', async ({ page }) => {
    await page.goto('/login');

    // Fill in credentials (mock gateway returns 401 for /api/auth/login)
    await page.fill('input#email', 'wrong@example.com');
    await page.fill('input#password', 'badpassword');

    const submitBtn = page.getByRole('button', { name: 'Sign in' });
    await submitBtn.dispatchEvent('click');

    // The submit handler calls fetch, which returns 401; error is shown
    await expect(page.getByText('Invalid credentials')).toBeVisible();
  });

  test('shows loading state while submitting', async ({ page }) => {
    await page.goto('/login');

    // Intercept the login POST and delay it
    await page.route('**/api/auth/login', async route => {
      await new Promise<void>(r => setTimeout(r, 400));
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Invalid credentials' }),
      }).catch(() => {});
    });

    await page.fill('input#email', 'test@example.com');
    await page.fill('input#password', 'password');

    const loginReq = page.waitForRequest('**/api/auth/login');
    await page.getByRole('button', { name: 'Sign in' }).dispatchEvent('click');
    await loginReq;

    await expect(page.getByRole('button', { name: 'Signing in…' })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Auth guard — unauthenticated redirect
// ---------------------------------------------------------------------------

test.describe('Auth guard redirects', () => {
  test('unauthenticated /host redirects to /login', async ({ page }) => {
    await page.goto('/host');
    await expect(page).toHaveURL(/\/login/);
  });

  test('unauthenticated /host/bookings redirects to /login', async ({ page }) => {
    await page.goto('/host/bookings');
    await expect(page).toHaveURL(/\/login/);
  });

  test('unauthenticated /host/listings/new redirects to /login', async ({ page }) => {
    await page.goto('/host/listings/new');
    await expect(page).toHaveURL(/\/login/);
  });
});
