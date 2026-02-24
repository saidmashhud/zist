/**
 * E2E tests — My trips page (/bookings)
 *
 * StatusBadge labels (from StatusBadge.svelte):
 *   pending_host_approval → "Awaiting approval"
 *   confirmed             → "Confirmed"
 *   cancelled_by_guest    → "Cancelled"
 */
import { test, expect } from './fixtures.js';

// ---------------------------------------------------------------------------
// Auth guard
// ---------------------------------------------------------------------------

test.describe('My trips — auth guard', () => {
  test('redirects unauthenticated users to /login', async ({ page }) => {
    await page.goto('/bookings');
    await expect(page).toHaveURL(/\/login/);
  });

  test('authenticated users can access /bookings', async ({ authedPage: page }) => {
    await page.goto('/bookings');
    await expect(page).toHaveURL(/\/bookings/);
    await expect(page.getByRole('heading', { name: 'My trips' })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Booking list
// ---------------------------------------------------------------------------

test.describe('My trips — booking list', () => {
  test('shows "Upcoming & active" and "Past & cancelled" sections', async ({ authedPage: page }) => {
    await page.goto('/bookings');

    await expect(page.getByRole('heading', { name: 'Upcoming & active' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Past & cancelled' })).toBeVisible();
  });

  test('pending_host_approval booking shows "Awaiting approval" badge', async ({ authedPage: page }) => {
    await page.goto('/bookings');

    const card = page.locator('a[href="/bookings/booking-001"]');
    await expect(card).toBeVisible();
    // StatusBadge for pending_host_approval = "Awaiting approval"
    await expect(card.getByText('Awaiting approval')).toBeVisible();
  });

  test('confirmed booking shows "Confirmed" badge', async ({ authedPage: page }) => {
    await page.goto('/bookings');

    const card = page.locator('a[href="/bookings/booking-002"]');
    await expect(card).toBeVisible();
    await expect(card.getByText('Confirmed')).toBeVisible();
  });

  test('cancelled booking appears in past section with "Cancelled" badge', async ({ authedPage: page }) => {
    await page.goto('/bookings');

    const card = page.locator('a[href="/bookings/booking-003"]');
    await expect(card).toBeVisible();
    await expect(card.getByText('Cancelled')).toBeVisible();
  });

  test('pending booking shows "Waiting for host" note', async ({ authedPage: page }) => {
    await page.goto('/bookings');

    await expect(page.getByText('Waiting for host to approve your request.')).toBeVisible();
  });

  test('shows formatted date range and night count', async ({ authedPage: page }) => {
    await page.goto('/bookings');

    // booking-001: 2026-03-10 → 2026-03-13 = 3 nights, 2 guests
    await expect(page.getByText(/3 nights/)).toBeVisible();
    await expect(page.getByText(/2 guests/)).toBeVisible();
  });

  test('empty state when no bookings (client-side navigation)', async ({ authedPage: page }) => {
    // Set up route before navigating so client-side load function is intercepted
    await page.route('**/api/bookings', async route => {
      if (route.request().method() !== 'GET') { await route.continue(); return; }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bookings: [] }) });
    });

    // Navigate home first, then click "My trips" → client-side load runs in browser
    await page.goto('/');
    await page.getByRole('link', { name: 'My trips' }).click();
    await page.waitForURL(/\/bookings/);

    await expect(page.getByText('No trips yet')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('link', { name: 'Explore stays →' })).toBeVisible();
  });

  test('?success=1 shows booking confirmed banner', async ({ authedPage: page }) => {
    await page.goto('/bookings?success=1');

    await expect(page.getByText('Your booking is confirmed. Enjoy your stay!')).toBeVisible();
  });

  test('clicking a booking card navigates to booking detail', async ({ authedPage: page }) => {
    await page.goto('/bookings');

    await page.locator('a[href="/bookings/booking-001"]').click();
    await page.waitForURL(/\/bookings\/booking-001/);
  });
});

// ---------------------------------------------------------------------------
// Booking detail page — /bookings/[id]
// ---------------------------------------------------------------------------

test.describe('Booking detail — /bookings/[id]', () => {
  test('shows booking status badge', async ({ authedPage: page }) => {
    await page.goto('/bookings/booking-001');

    await expect(page.getByText('Awaiting approval')).toBeVisible();
  });

  test('shows formatted dates', async ({ authedPage: page }) => {
    await page.goto('/bookings/booking-001');

    // fmtDate('2026-03-10') → shows formatted date
    await expect(page.getByText(/Mar/)).toBeVisible();
  });

  test('shows guest count', async ({ authedPage: page }) => {
    await page.goto('/bookings/booking-001');

    await expect(page.getByText('2 guests')).toBeVisible();
  });

  test('shows listing title in detail', async ({ authedPage: page }) => {
    await page.goto('/bookings/booking-001');

    // load function fetches listing-001 → "Cozy Apartment in Tashkent"
    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
  });

  test('shows total amount', async ({ authedPage: page }) => {
    await page.goto('/bookings/booking-001');

    // 513000 formatted as 513,000 (toLocaleString)
    await expect(page.getByText(/513,?000/)).toBeVisible();
  });

  test('shows cancel button for active (pending) booking', async ({ authedPage: page }) => {
    await page.goto('/bookings/booking-001');

    await expect(page.getByRole('button', { name: 'Cancel booking' })).toBeVisible();
  });

  test('cancel booking sends POST after confirm dialog', async ({ authedPage: page }) => {
    await page.goto('/bookings/booking-001');

    let cancelCalled = false;
    await page.route('**/api/bookings/booking-001/cancel', async route => {
      cancelCalled = true;
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ status: 'cancelled_by_guest' }),
      });
    });

    // page.once('dialog', …) handles the dialog inline during dispatchEvent,
    // avoiding the deadlock that occurs with page.waitForEvent('dialog').
    let dialogMessage = '';
    page.once('dialog', async dialog => {
      dialogMessage = dialog.message();
      await dialog.accept();
    });
    const cancelReq = page.waitForRequest('**/api/bookings/booking-001/cancel');
    await page.getByRole('button', { name: 'Cancel booking' }).dispatchEvent('click');
    await cancelReq;

    expect(dialogMessage).toContain('Are you sure you want to cancel');
    expect(cancelCalled).toBe(true);
  });

  test('cancel button not shown for cancelled booking', async ({ authedPage: page }) => {
    // booking-003 is cancelled_by_guest — canCancel = false
    await page.goto('/bookings/booking-003');

    await expect(page.getByRole('button', { name: 'Cancel booking' })).not.toBeVisible();
  });
});
