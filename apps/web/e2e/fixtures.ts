/**
 * Custom Playwright test fixtures for Zist E2E tests.
 *
 * Provides:
 *   - `authedPage`  — a Page with the guest session cookie pre-set
 *   - `hostPage`    — a Page with the host session cookie pre-set
 *   - `makeSessionCookie` — helper to craft a fake zist_session JWT
 *
 * The JWT is NOT signature-verified by hooks.server.ts; it only checks that
 * the base64url-decoded payload has a valid `exp` field. So a hand-crafted
 * token with `fakesig` as the signature works perfectly in tests.
 */
import { test as base, expect, type Page } from '@playwright/test';
import { MOCK_GUEST, MOCK_HOST } from './mock-data.js';

export { expect };

// ---------------------------------------------------------------------------
// JWT cookie factory
// ---------------------------------------------------------------------------

function toBase64Url(str: string): string {
  return Buffer.from(str)
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');
}

export function makeSessionCookie(payload: object): string {
  const header = toBase64Url(JSON.stringify({ alg: 'RS256', typ: 'JWT' }));
  const body = toBase64Url(JSON.stringify(payload));
  return `${header}.${body}.fakesig`;
}

export const GUEST_SESSION = makeSessionCookie(MOCK_GUEST);
export const HOST_SESSION = makeSessionCookie(MOCK_HOST);

// ---------------------------------------------------------------------------
// Fixture types
// ---------------------------------------------------------------------------

type Fixtures = {
  /** Authenticated as the test guest user. */
  authedPage: Page;
  /** Authenticated as the test host user. */
  hostPage: Page;
};

// ---------------------------------------------------------------------------
// Extended test object
// ---------------------------------------------------------------------------

export const test = base.extend<Fixtures>({
  // Override page.goto to wait for networkidle by default.
  // Svelte 5 in Vite dev mode loads JS modules lazily, so event delegation
  // listeners are not set up until after the 'load' event fires. Waiting for
  // networkidle ensures full hydration before any button interactions.
  page: async ({ page }, use) => {
    const originalGoto = page.goto.bind(page);
    (page as any).goto = (url: string, options?: Parameters<typeof page.goto>[1]) =>
      originalGoto(url, { waitUntil: 'networkidle', ...options });
    await use(page);
  },

  authedPage: async ({ page, context }, use) => {
    await context.addCookies([
      {
        name: 'zist_session',
        value: GUEST_SESSION,
        domain: 'localhost',
        path: '/',
      },
    ]);
    await use(page);
  },

  hostPage: async ({ page, context }, use) => {
    await context.addCookies([
      {
        name: 'zist_session',
        value: HOST_SESSION,
        domain: 'localhost',
        path: '/',
      },
    ]);
    await use(page);
  },
});

// ---------------------------------------------------------------------------
// Shared route helpers
// ---------------------------------------------------------------------------

/**
 * Intercepts a single API call and fulfills it with the given JSON body.
 * Returns after the first matching request.
 */
export async function mockOnce(
  page: Page,
  urlPattern: string | RegExp,
  status: number,
  body: unknown
): Promise<void> {
  await page.route(urlPattern, async route => {
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(body),
    });
    // Unroute after first match so subsequent requests go to the real mock gateway
    await page.unroute(urlPattern);
  });
}

/**
 * Intercepts an API call once, captures the request, and delegates to mock gateway.
 * Returns the captured request body.
 */
export function captureRequest(
  page: Page,
  urlPattern: string | RegExp,
  method = 'POST'
): Promise<{ url: string; body: unknown }> {
  return new Promise((resolve, reject) => {
    page.route(urlPattern, async route => {
      const req = route.request();
      if (req.method() !== method) {
        await route.continue();
        return;
      }
      let body: unknown = null;
      try {
        body = JSON.parse(req.postData() ?? 'null');
      } catch { /* non-JSON body */ }
      // Let the mock gateway handle it
      await route.continue();
      resolve({ url: req.url(), body });
    }).catch(reject);
  });
}
