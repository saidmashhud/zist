/**
 * E2E tests — Admin webhook endpoints (/admin/webhooks)
 *
 * The admin page uses Svelte 4 `on:click` event directives (addEventListener),
 * loads data via onMount, and has no server-side auth guard.
 */
import { test, expect } from './fixtures.js';
import { WEBHOOK_ENDPOINT_STUB } from './mock-data.js';

// ---------------------------------------------------------------------------
// Page load
// ---------------------------------------------------------------------------

test.describe('Admin webhooks — page load', () => {
  test('shows "Webhook Endpoints" heading', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    await expect(page.getByRole('heading', { name: 'Webhook Endpoints' })).toBeVisible();
  });

  test('shows existing endpoint URL from stub', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    // Wait for onMount fetch to complete and render endpoint
    await expect(page.getByText(WEBHOOK_ENDPOINT_STUB.url)).toBeVisible();
  });

  test('shows endpoint status badge', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    await expect(page.getByText('active')).toBeVisible();
  });

  test('shows event types for the existing endpoint', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    await expect(page.getByText('booking.created')).toBeVisible();
    await expect(page.getByText('booking.confirmed')).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Create endpoint
// ---------------------------------------------------------------------------

test.describe('Admin webhooks — create endpoint', () => {
  test('clicking "+ New Endpoint" shows the create form', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    await page.getByRole('button', { name: '+ New Endpoint' }).dispatchEvent('click');

    await expect(page.getByRole('heading', { name: 'Create Endpoint' })).toBeVisible();
    await expect(page.getByPlaceholder('https://your-app.com/webhooks/mashgate')).toBeVisible();
  });

  test('Cancel button hides the create form', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    await page.getByRole('button', { name: '+ New Endpoint' }).dispatchEvent('click');
    await expect(page.getByRole('heading', { name: 'Create Endpoint' })).toBeVisible();

    await page.getByRole('button', { name: 'Cancel' }).dispatchEvent('click');
    await expect(page.getByRole('heading', { name: 'Create Endpoint' })).not.toBeVisible();
  });

  test('Create button is disabled when URL is empty', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    await page.getByRole('button', { name: '+ New Endpoint' }).dispatchEvent('click');

    const createBtn = page.getByRole('button', { name: 'Create' });
    await expect(createBtn).toBeDisabled();
  });

  test('filling URL and clicking Create sends POST to /api/admin/webhooks/endpoints', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    await page.getByRole('button', { name: '+ New Endpoint' }).dispatchEvent('click');

    await page.fill('input[type="url"]', 'https://my-app.example.com/hooks');
    await page.locator('input[class*="font-mono"]').fill('payment.captured');

    const postReq = page.waitForRequest(
      req => req.url().includes('/api/admin/webhooks/endpoints') && req.method() === 'POST'
    );
    await page.getByRole('button', { name: 'Create' }).dispatchEvent('click');
    const req = await postReq;

    // Verify request body contains the URL we filled in
    const body = JSON.parse(req.postData() ?? '{}');
    expect(body).toMatchObject({ url: 'https://my-app.example.com/hooks' });
  });

  test('after creation the new endpoint URL appears in the list', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');

    const newEndpoint = {
      ...WEBHOOK_ENDPOINT_STUB,
      id: 'wh-002',
      url: 'https://new-endpoint.example.com/hook',
      eventTypes: ['booking.created'],
    };

    await page.route('**/api/admin/webhooks/endpoints', async route => {
      if (route.request().method() !== 'POST') { await route.continue(); return; }
      await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(newEndpoint) });
    });

    await page.getByRole('button', { name: '+ New Endpoint' }).dispatchEvent('click');
    await page.fill('input[type="url"]', 'https://new-endpoint.example.com/hook');

    const postReq = page.waitForRequest(
      req => req.url().includes('/api/admin/webhooks/endpoints') && req.method() === 'POST'
    );
    await page.getByRole('button', { name: 'Create' }).dispatchEvent('click');
    await postReq;

    await expect(page.getByText('https://new-endpoint.example.com/hook')).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Delete endpoint
// ---------------------------------------------------------------------------

test.describe('Admin webhooks — delete endpoint', () => {
  test('Delete button shows confirm dialog', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');
    await expect(page.getByText(WEBHOOK_ENDPOINT_STUB.url)).toBeVisible();

    let dialogMessage = '';
    page.once('dialog', async d => {
      dialogMessage = d.message();
      await d.dismiss();
    });
    await page.getByRole('button', { name: 'Delete' }).first().dispatchEvent('click');

    expect(dialogMessage).toContain('Delete this endpoint?');
  });

  test('confirming delete sends DELETE to /api/admin/webhooks/endpoints/wh-001', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');
    await expect(page.getByText(WEBHOOK_ENDPOINT_STUB.url)).toBeVisible();

    const deleteReq = page.waitForRequest(
      req => req.url().includes('/api/admin/webhooks/endpoints/') && req.method() === 'DELETE'
    );
    page.once('dialog', async d => { await d.accept(); });
    await page.getByRole('button', { name: 'Delete' }).first().dispatchEvent('click');
    const req = await deleteReq;

    expect(req.url()).toContain('/api/admin/webhooks/endpoints/wh-001');
  });

  test('endpoint is removed from list after deletion', async ({ hostPage: page }) => {
    await page.goto('/admin/webhooks');
    await expect(page.getByText(WEBHOOK_ENDPOINT_STUB.url)).toBeVisible();

    await page.route('**/api/admin/webhooks/endpoints/**', async route => {
      if (route.request().method() !== 'DELETE') { await route.continue(); return; }
      await route.fulfill({ status: 204 });
    });

    const deleteReq = page.waitForRequest(
      req => req.url().includes('/api/admin/webhooks/endpoints/') && req.method() === 'DELETE'
    );
    page.once('dialog', async d => { await d.accept(); });
    await page.getByRole('button', { name: 'Delete' }).first().dispatchEvent('click');
    await deleteReq;

    // After deletion, the endpoint URL should be gone from the list
    await expect(page.getByText(WEBHOOK_ENDPOINT_STUB.url)).not.toBeVisible();
  });
});
