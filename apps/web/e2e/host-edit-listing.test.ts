/**
 * E2E tests — Edit listing (/host/listings/[id]/edit)
 */
import { test, expect } from './fixtures.js';
import { LISTING_TASHKENT, LISTING_PAUSED } from './mock-data.js';

// ---------------------------------------------------------------------------
// Auth guard
// ---------------------------------------------------------------------------

test.describe('Edit listing — auth guard', () => {
  test('unauthenticated access redirects to /login', async ({ page }) => {
    await page.goto('/host/listings/listing-001/edit');
    await expect(page).toHaveURL(/\/login/);
  });
});

// ---------------------------------------------------------------------------
// Load / display
// ---------------------------------------------------------------------------

test.describe('Edit listing — load', () => {
  test('loads and shows pre-filled listing title', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit');

    // h1 shows the listing title
    await expect(page.getByRole('heading', { name: LISTING_TASHKENT.title })).toBeVisible();
    // title input is pre-filled
    await expect(page.locator('input#edit-title')).toHaveValue(LISTING_TASHKENT.title);
  });

  test('?created=1 shows "Listing created!" banner', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit?created=1');

    await expect(page.getByText('Listing created!')).toBeVisible();
  });

  test('shows "Pause listing" button for active listing', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit');

    await expect(page.getByRole('button', { name: 'Pause listing' })).toBeVisible();
  });

  test('shows "Publish listing" button for paused listing', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-003/edit');

    await expect(page.getByRole('button', { name: 'Publish listing' })).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

test.describe('Edit listing — save', () => {
  test('Save changes sends PUT to /api/listings/listing-001', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit');

    let putCalled = false;
    await page.route('**/api/listings/listing-001', async route => {
      if (route.request().method() !== 'PUT') { await route.continue(); return; }
      putCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...LISTING_TASHKENT, title: 'Updated Title' }),
      });
    });

    // Change title
    await page.fill('#edit-title', 'Updated Title');

    const putReq = page.waitForRequest(
      req => req.url().includes('/api/listings/listing-001') && req.method() === 'PUT'
    );
    await page.getByRole('button', { name: 'Save changes' }).dispatchEvent('click');
    await putReq;

    expect(putCalled).toBe(true);
  });

  test('successful save shows "Changes saved." banner', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit');

    await page.route('**/api/listings/listing-001', async route => {
      if (route.request().method() !== 'PUT') { await route.continue(); return; }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(LISTING_TASHKENT),
      });
    });

    const putReq = page.waitForRequest(
      req => req.url().includes('/api/listings/listing-001') && req.method() === 'PUT'
    );
    await page.getByRole('button', { name: 'Save changes' }).dispatchEvent('click');
    await putReq;

    await expect(page.getByText('Changes saved.')).toBeVisible();
  });

  test('PUT body contains updated title', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit');

    let capturedBody: Record<string, unknown> = {};
    await page.route('**/api/listings/listing-001', async route => {
      if (route.request().method() !== 'PUT') { await route.continue(); return; }
      try { capturedBody = JSON.parse(route.request().postData() ?? '{}'); } catch { /* */ }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(LISTING_TASHKENT),
      });
    });

    await page.fill('#edit-title', 'Brand New Title');

    const putReq = page.waitForRequest(
      req => req.url().includes('/api/listings/listing-001') && req.method() === 'PUT'
    );
    await page.getByRole('button', { name: 'Save changes' }).dispatchEvent('click');
    await putReq;

    expect(capturedBody).toMatchObject({ title: 'Brand New Title' });
  });
});

// ---------------------------------------------------------------------------
// Publish / Unpublish
// ---------------------------------------------------------------------------

test.describe('Edit listing — publish/unpublish', () => {
  test('Pause listing sends POST to /api/listings/listing-001/unpublish', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit');

    let unpublishCalled = false;
    await page.route('**/api/listings/listing-001/unpublish', async route => {
      unpublishCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...LISTING_TASHKENT, status: 'paused' }),
      });
    });
    // Intercept the host dashboard load after goto('/host')
    await page.route('**/api/listings/mine', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ listings: [] }) });
    });
    await page.route('**/api/bookings/host', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bookings: [] }) });
    });

    const unpublishReq = page.waitForRequest('**/api/listings/listing-001/unpublish');
    await page.getByRole('button', { name: 'Pause listing' }).dispatchEvent('click');
    await unpublishReq;

    expect(unpublishCalled).toBe(true);
    await page.waitForURL(/\/host$/);
  });

  test('Publish listing sends POST to /api/listings/listing-003/publish', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-003/edit');

    let publishCalled = false;
    await page.route('**/api/listings/listing-003/publish', async route => {
      publishCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...LISTING_PAUSED, status: 'active' }),
      });
    });
    await page.route('**/api/listings/mine', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ listings: [] }) });
    });
    await page.route('**/api/bookings/host', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bookings: [] }) });
    });

    const publishReq = page.waitForRequest('**/api/listings/listing-003/publish');
    await page.getByRole('button', { name: 'Publish listing' }).dispatchEvent('click');
    await publishReq;

    expect(publishCalled).toBe(true);
    await page.waitForURL(/\/host$/);
  });
});

// ---------------------------------------------------------------------------
// Photos
// ---------------------------------------------------------------------------

test.describe('Edit listing — photos', () => {
  test('Add photo button sends POST to /api/listings/listing-001/photos', async ({ hostPage: page }) => {
    await page.goto('/host/listings/listing-001/edit');

    await page.fill('input[type="url"]', 'https://example.com/photo1.jpg');

    // waitForRequest resolves when the browser makes the request; postData() is available on it
    const photoReq = page.waitForRequest(
      req => req.url().includes('/api/listings/listing-001/photos') && req.method() === 'POST'
    );
    await page.getByRole('button', { name: 'Add' }).dispatchEvent('click');
    const req = await photoReq;

    expect(req.url()).toContain('/api/listings/listing-001/photos');
  });

  test('Delete photo sends DELETE to /api/listings/listing-001/photos/:id', async ({ hostPage: page }) => {
    // LISTING_TASHKENT in mock-data.ts has photo-001 pre-loaded; the SSR load will return it
    await page.goto('/host/listings/listing-001/edit');

    // Verify the photo image is rendered (listing-001 has a photo in mock data)
    await expect(page.locator('.relative.group').first()).toBeVisible();

    let deletePhotoUrl = '';
    await page.route('**/api/listings/listing-001/photos/**', async route => {
      if (route.request().method() !== 'DELETE') { await route.continue(); return; }
      deletePhotoUrl = route.request().url();
      await route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ ...LISTING_TASHKENT, photos: [] }) });
    });

    const photoContainer = page.locator('.relative.group').first();
    await photoContainer.hover();

    const deleteReq = page.waitForRequest(
      req => req.url().includes('/api/listings/listing-001/photos/') && req.method() === 'DELETE'
    );

    // deletePhoto uses confirm(); set handler before dispatchEvent
    page.once('dialog', async d => { await d.accept(); });
    await photoContainer.locator('button').dispatchEvent('click');
    await deleteReq;

    expect(deletePhotoUrl).toContain('/api/listings/listing-001/photos/photo-001');
  });
});
