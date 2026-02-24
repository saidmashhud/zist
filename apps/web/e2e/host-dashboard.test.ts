/**
 * E2E tests — Host dashboard (/host)
 *
 * StatusBadge labels:
 *   pending_host_approval → "Awaiting approval"
 *   confirmed             → "Confirmed"
 */
import { test, expect } from './fixtures.js';
import { HOST_OWN_LISTINGS, HOST_BOOKINGS, BOOKING_PENDING } from './mock-data.js';

// ---------------------------------------------------------------------------
// Auth guard
// ---------------------------------------------------------------------------

test.describe('Host dashboard — auth guard', () => {
  test('redirects unauthenticated users to /login', async ({ page }) => {
    await page.goto('/host');
    await expect(page).toHaveURL(/\/login/);
  });

  test('host can access /host', async ({ hostPage: page }) => {
    await page.goto('/host');
    await expect(page).toHaveURL(/\/host/);
    await expect(page.getByRole('heading', { name: 'Host dashboard' })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Stats row
// ---------------------------------------------------------------------------

test.describe('Host dashboard — stats', () => {
  test('shows active listings count with label', async ({ hostPage: page }) => {
    await page.goto('/host');

    // Find the stat card whose label says "Active listings"
    const card = page.locator('.rounded-2xl.border').filter({ hasText: 'Active listings' });
    const activeCount = HOST_OWN_LISTINGS.filter(l => l.status === 'active').length;
    await expect(card.getByText(String(activeCount))).toBeVisible();
  });

  test('shows pending bookings count', async ({ hostPage: page }) => {
    await page.goto('/host');

    const card = page.locator('.rounded-2xl.border').filter({ hasText: 'Awaiting approval' }).first();
    const pendingCount = HOST_BOOKINGS.filter(b => b.status === 'pending_host_approval').length;
    await expect(card.getByText(String(pendingCount))).toBeVisible();
  });

  test('shows confirmed bookings count', async ({ hostPage: page }) => {
    await page.goto('/host');

    const card = page.locator('.rounded-2xl.border').filter({ hasText: 'Confirmed bookings' });
    const confirmedCount = HOST_BOOKINGS.filter(
      b => b.status === 'confirmed' || b.status === 'payment_pending'
    ).length;
    await expect(card.getByText(String(confirmedCount))).toBeVisible();
  });

  test('shows total listings count', async ({ hostPage: page }) => {
    await page.goto('/host');

    const card = page.locator('.rounded-2xl.border').filter({ hasText: 'Total listings' });
    await expect(card.getByText(String(HOST_OWN_LISTINGS.length))).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Action required — pending bookings
// ---------------------------------------------------------------------------

test.describe('Host dashboard — pending booking actions', () => {
  test('"Action required" section shows for pending bookings', async ({ hostPage: page }) => {
    await page.goto('/host');

    await expect(page.getByRole('heading', { name: 'Action required' })).toBeVisible();
    await expect(page.getByText('Looking forward to the stay!')).toBeVisible();
  });

  test('Approve button sends POST to /api/bookings/{id}/approve', async ({ hostPage: page }) => {
    await page.goto('/host');

    let approveUrl = '';
    await page.route('**/api/bookings/*/approve', async route => {
      approveUrl = route.request().url();
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ ...BOOKING_PENDING, status: 'payment_pending' }),
      });
    });

    // Start listening before the click so the request isn't missed
    const approveReq = page.waitForRequest('**/api/bookings/*/approve');
    await page.getByRole('button', { name: 'Approve' }).first().dispatchEvent('click');
    await approveReq;

    expect(approveUrl).toContain('/api/bookings/booking-001/approve');
  });

  test('Approve button shows "…" loading state', async ({ hostPage: page }) => {
    await page.goto('/host');

    // Use a real setTimeout (not page.waitForTimeout) so the route callback
    // doesn't throw "Test ended" when the test completes before the 300ms delay.
    await page.route('**/api/bookings/*/approve', async route => {
      await new Promise<void>(r => setTimeout(r, 300));
      await route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ ...BOOKING_PENDING, status: 'payment_pending' }) }).catch(() => {});
    });

    // Capture the request before clicking so it isn't missed
    const approveReq = page.waitForRequest('**/api/bookings/*/approve');
    await page.getByRole('button', { name: 'Approve' }).first().dispatchEvent('click');
    await approveReq; // request is in-flight; response is delayed 300 ms
    await expect(page.getByRole('button', { name: '…' })).toBeVisible();
  });

  test('Decline button shows confirm dialog before sending', async ({ hostPage: page }) => {
    await page.goto('/host');

    // page.waitForEvent('dialog') prevents auto-dismiss, causing a deadlock with dispatchEvent.
    // Use page.once('dialog', …) instead: the handler runs inline during dispatchEvent,
    // dismisses the dialog, and lets dispatchEvent complete.
    let dialogMessage = '';
    page.once('dialog', async dialog => {
      dialogMessage = dialog.message();
      await dialog.dismiss();
    });
    await page.getByRole('button', { name: 'Decline' }).first().dispatchEvent('click');

    expect(dialogMessage).toContain('Decline this booking request?');
  });

  test('Decline confirmed sends POST to reject endpoint', async ({ hostPage: page }) => {
    await page.goto('/host');

    let rejectCalled = false;
    await page.route('**/api/bookings/*/reject', async route => {
      rejectCalled = true;
      await route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ ...BOOKING_PENDING, status: 'rejected' }) });
    });

    const rejectReq = page.waitForRequest('**/api/bookings/*/reject');
    page.once('dialog', d => d.accept());
    await page.getByRole('button', { name: 'Decline' }).first().dispatchEvent('click');
    await rejectReq;

    expect(rejectCalled).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// Recent bookings list
// ---------------------------------------------------------------------------

test.describe('Host dashboard — recent bookings', () => {
  test('shows recent bookings with status badges', async ({ hostPage: page }) => {
    await page.goto('/host');

    await expect(page.getByRole('heading', { name: 'Recent bookings' })).toBeVisible();
    // STATUS badges appear in the recent booking rows
    await expect(page.getByText('Awaiting approval').first()).toBeVisible();
    await expect(page.getByText('Confirmed').first()).toBeVisible();
  });

  test('"View all" link navigates to /host/bookings', async ({ hostPage: page }) => {
    await page.goto('/host');

    await page.getByRole('link', { name: 'View all' }).click();
    await page.waitForURL(/\/host\/bookings/);
  });

  test('clicking a recent booking row navigates to booking detail', async ({ hostPage: page }) => {
    await page.goto('/host');

    await page.locator('a[href="/host/bookings/booking-001"]').first().click();
    await page.waitForURL(/\/host\/bookings\/booking-001/);
  });
});

// ---------------------------------------------------------------------------
// My listings panel
// ---------------------------------------------------------------------------

test.describe('Host dashboard — my listings', () => {
  test('shows host listings', async ({ hostPage: page }) => {
    await page.goto('/host');

    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
    await expect(page.getByText('Samarkand Historical Suite')).toBeVisible();
    await expect(page.getByText('Bukhara Silk Road Retreat')).toBeVisible();
    await expect(page.getByText('Tashkent Modern Room')).toBeVisible();
    await expect(page.getByText('Fergana Valley House')).toBeVisible();
  });

  test('listing shows active/paused status', async ({ hostPage: page }) => {
    await page.goto('/host');

    await expect(page.getByText('active', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('paused', { exact: true })).toBeVisible();
  });

  test('Edit link points to /host/listings/[id]/edit', async ({ hostPage: page }) => {
    await page.goto('/host');

    await expect(page.locator('a[href="/host/listings/listing-001/edit"]')).toBeVisible();
  });

  test('View link navigates to public listing page', async ({ hostPage: page }) => {
    await page.goto('/host');

    await page.locator('a[href="/listings/listing-001"]').first().click();
    await page.waitForURL(/\/listings\/listing-001/);
    await expect(page.getByRole('heading', { name: 'Cozy Apartment in Tashkent' })).toBeVisible();
  });

  test('Pause button for active listing sends PATCH', async ({ hostPage: page }) => {
    await page.goto('/host');

    let patchBody: unknown = null;
    await page.route('**/api/listings/listing-001', async route => {
      if (route.request().method() !== 'PATCH') { await route.continue(); return; }
      patchBody = route.request().postDataJSON();
      await route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ ...HOST_OWN_LISTINGS[0], status: 'paused' }) });
    });

    await page.getByRole('button', { name: 'Pause' }).first().dispatchEvent('click');
    await page.waitForTimeout(500);

    expect(patchBody).toMatchObject({ status: 'paused' });
  });

  test('Activate button for paused listing sends PATCH with active', async ({ hostPage: page }) => {
    await page.goto('/host');

    let patchBody: unknown = null;
    await page.route('**/api/listings/listing-003', async route => {
      if (route.request().method() !== 'PATCH') { await route.continue(); return; }
      patchBody = route.request().postDataJSON();
      await route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ ...HOST_OWN_LISTINGS[2], status: 'active' }) });
    });

    await page.getByRole('button', { name: 'Activate' }).first().dispatchEvent('click');
    await page.waitForTimeout(500);

    expect(patchBody).toMatchObject({ status: 'active' });
  });

  test('"New listing" button links to /host/listings/new', async ({ hostPage: page }) => {
    await page.goto('/host');

    await expect(page.getByRole('link', { name: '+ New listing' })).toHaveAttribute('href', '/host/listings/new');
  });

  test('"Manage" link goes to /host/listings', async ({ hostPage: page }) => {
    await page.goto('/host');

    await expect(page.getByRole('link', { name: 'Manage' })).toHaveAttribute('href', '/host/listings');
  });
});

// ---------------------------------------------------------------------------
// Host bookings list page (/host/bookings)
// ---------------------------------------------------------------------------

test.describe('Host bookings page', () => {
  test('shows all host bookings with status badges', async ({ hostPage: page }) => {
    await page.goto('/host/bookings');

    await expect(page.getByText('Awaiting approval').first()).toBeVisible();
    await expect(page.getByText('Confirmed').first()).toBeVisible();
  });

  test('redirects to /login when unauthenticated', async ({ page }) => {
    await page.goto('/host/bookings');
    await expect(page).toHaveURL(/\/login/);
  });
});

// ---------------------------------------------------------------------------
// Host booking detail page (/host/bookings/[id])
// ---------------------------------------------------------------------------

test.describe('Host booking detail', () => {
  test('shows booking info: listing title, dates, guests, status badge', async ({ hostPage: page }) => {
    await page.goto('/host/bookings/booking-001');

    await expect(page.getByText('Cozy Apartment in Tashkent')).toBeVisible();
    // fmtDate outputs formatted date, check for month abbreviation
    await expect(page.getByText(/Mar/)).toBeVisible();
    await expect(page.getByText('2 guests').first()).toBeVisible();
    await expect(page.getByText('Awaiting approval')).toBeVisible();
  });

  test('shows Approve and Decline buttons for pending booking', async ({ hostPage: page }) => {
    await page.goto('/host/bookings/booking-001');

    await expect(page.getByRole('button', { name: 'Approve' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Decline' })).toBeVisible();
  });

  test('no Approve/Decline for confirmed booking', async ({ hostPage: page }) => {
    await page.goto('/host/bookings/booking-002');

    await expect(page.getByRole('button', { name: 'Approve' })).not.toBeVisible();
    await expect(page.getByRole('button', { name: 'Decline' })).not.toBeVisible();
  });

  test('Decline button on detail page shows confirm dialog', async ({ hostPage: page }) => {
    await page.goto('/host/bookings/booking-001');

    let dialogMessage = '';
    page.once('dialog', async dialog => {
      dialogMessage = dialog.message();
      await dialog.dismiss();
    });
    await page.getByRole('button', { name: 'Decline' }).dispatchEvent('click');

    expect(dialogMessage).toContain('Decline this booking request?');
  });

  test('redirects to /login when unauthenticated', async ({ page }) => {
    await page.goto('/host/bookings/booking-001');
    await expect(page).toHaveURL(/\/login/);
  });
});
