/**
 * Lightweight mock HTTP gateway for Playwright E2E tests.
 *
 * Mimics the real gateway's API surface so SvelteKit's SSR load functions
 * and browser-side fetches both get deterministic test data.
 *
 * Started in global-setup.ts; all /api/* requests are routed here via the
 * Vite dev server proxy (GATEWAY_URL=http://localhost:9999).
 */
import http from 'http';
import type { IncomingMessage, ServerResponse } from 'http';
import {
  ALL_LISTINGS,
  HOST_OWN_LISTINGS,
  LISTING_TASHKENT,
  GUEST_BOOKINGS,
  HOST_BOOKINGS,
  BOOKING_PENDING,
  PRICE_PREVIEW_3N,
  PRICE_PREVIEW_4N,
  CHECKOUT_SESSION,
  WEBHOOK_ENDPOINT_STUB,
} from './mock-data.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function json(res: ServerResponse, status: number, data: unknown): void {
  const body = JSON.stringify(data);
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(body),
    'Access-Control-Allow-Origin': '*',
  });
  res.end(body);
}

function readBody(req: IncomingMessage): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    let raw = '';
    req.on('data', chunk => { raw += chunk; });
    req.on('end', () => {
      try { resolve(JSON.parse(raw || '{}')); } catch { resolve({}); }
    });
    req.on('error', reject);
  });
}

function notFound(res: ServerResponse): void {
  json(res, 404, { error: 'not found' });
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

function handle(req: IncomingMessage, res: ServerResponse): void {
  const rawUrl = req.url ?? '/';
  const { pathname } = new URL(rawUrl, 'http://localhost');
  const method = req.method ?? 'GET';

  // CORS preflight
  if (method === 'OPTIONS') {
    res.writeHead(204, {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET,POST,PUT,PATCH,DELETE',
      'Access-Control-Allow-Headers': 'Content-Type',
    });
    res.end();
    return;
  }

  // ── Listings ──────────────────────────────────────────────────────────────

  // GET /api/listings/mine
  if (method === 'GET' && pathname === '/api/listings/mine') {
    return json(res, 200, { listings: HOST_OWN_LISTINGS });
  }

  // GET /api/listings/search
  if (method === 'GET' && pathname === '/api/listings/search') {
    const qp = new URL(rawUrl, 'http://localhost').searchParams;
    const city = qp.get('city');
    const instant = qp.get('instant_book');
    const type = qp.get('type');
    let results = [...ALL_LISTINGS];
    if (city) results = results.filter(l => l.city.toLowerCase().includes(city.toLowerCase()));
    if (instant === 'true') results = results.filter(l => l.instantBook);
    if (type) results = results.filter(l => l.type === type);
    return json(res, 200, { listings: results, total: results.length });
  }

  // POST /api/listings (create)
  if (method === 'POST' && pathname === '/api/listings') {
    readBody(req).then(body => {
      const newListing = {
        ...LISTING_TASHKENT,
        id: 'listing-new-001',
        title: body.title ?? 'New Listing',
        description: body.description ?? '',
        city: body.city ?? 'Tashkent',
        country: body.country ?? 'Uzbekistan',
        address: body.address ?? '',
        type: body.type ?? 'apartment',
        pricePerNight: body.pricePerNight ?? '0',
        currency: body.currency ?? 'USD',
        instantBook: body.instantBook ?? false,
        amenities: body.amenities ?? [],
        status: 'draft',
      };
      json(res, 201, newListing);
    });
    return;
  }

  // GET /api/listings (featured)
  if (method === 'GET' && pathname === '/api/listings') {
    return json(res, 200, { listings: ALL_LISTINGS });
  }

  // GET /api/listings/:id/calendar
  const calendarMatch = pathname.match(/^\/api\/listings\/([^/]+)\/calendar$/);
  if (method === 'GET' && calendarMatch) {
    return json(res, 200, { days: [] });
  }

  // GET /api/listings/:id/price-preview
  const previewMatch = pathname.match(/^\/api\/listings\/([^/]+)\/price-preview$/);
  if (method === 'GET' && previewMatch) {
    const qp = new URL(rawUrl, 'http://localhost').searchParams;
    const checkIn = qp.get('check_in') ?? qp.get('checkIn') ?? '';
    const checkOut = qp.get('check_out') ?? qp.get('checkOut') ?? '';
    const nights = checkIn && checkOut
      ? Math.max(1, Math.round((new Date(checkOut).getTime() - new Date(checkIn).getTime()) / 86_400_000))
      : 3;
    const preview = nights >= 4 ? PRICE_PREVIEW_4N : PRICE_PREVIEW_3N;
    return json(res, 200, { ...preview, nights });
  }

  // POST /api/listings/:id/publish
  const publishMatch = pathname.match(/^\/api\/listings\/([^/]+)\/publish$/);
  if (method === 'POST' && publishMatch) {
    const id = publishMatch[1];
    const listing = ALL_LISTINGS.find(l => l.id === id);
    if (!listing) return notFound(res);
    return json(res, 200, { ...listing, status: 'active' });
  }

  // POST /api/listings/:id/unpublish
  const unpublishMatch = pathname.match(/^\/api\/listings\/([^/]+)\/unpublish$/);
  if (method === 'POST' && unpublishMatch) {
    const id = unpublishMatch[1];
    const listing = ALL_LISTINGS.find(l => l.id === id);
    if (!listing) return notFound(res);
    return json(res, 200, { ...listing, status: 'paused' });
  }

  // POST /api/listings/:id/photos
  const photosMatch = pathname.match(/^\/api\/listings\/([^/]+)\/photos$/);
  if (method === 'POST' && photosMatch) {
    const id = photosMatch[1];
    const listing = ALL_LISTINGS.find(l => l.id === id);
    if (!listing) return notFound(res);
    readBody(req).then(body => {
      const photo = { id: 'photo-new-001', url: body.url ?? '', caption: body.caption ?? '' };
      json(res, 200, { ...listing, photos: [...listing.photos, photo] });
    });
    return;
  }

  // DELETE /api/listings/:id/photos/:photoId
  const photoDeleteMatch = pathname.match(/^\/api\/listings\/([^/]+)\/photos\/([^/]+)$/);
  if (method === 'DELETE' && photoDeleteMatch) {
    const id = photoDeleteMatch[1];
    const listing = ALL_LISTINGS.find(l => l.id === id);
    if (!listing) return notFound(res);
    return json(res, 200, { ...listing, photos: [] });
  }

  // PATCH /api/listings/:id  — merges body
  // PUT   /api/listings/:id  — full replace (same behaviour in mock)
  const listingIdMatch = pathname.match(/^\/api\/listings\/([^/]+)$/);
  if ((method === 'PATCH' || method === 'PUT') && listingIdMatch) {
    const id = listingIdMatch[1];
    const listing = ALL_LISTINGS.find(l => l.id === id);
    if (!listing) return notFound(res);
    readBody(req).then(body => {
      json(res, 200, { ...listing, ...body, id });
    });
    return;
  }

  // GET /api/listings/:id
  if (method === 'GET' && listingIdMatch) {
    const id = listingIdMatch[1];
    const listing = ALL_LISTINGS.find(l => l.id === id);
    if (!listing) return notFound(res);
    return json(res, 200, listing);
  }

  // ── Bookings ──────────────────────────────────────────────────────────────

  // GET /api/bookings/host
  if (method === 'GET' && pathname === '/api/bookings/host') {
    return json(res, 200, { bookings: HOST_BOOKINGS });
  }

  // POST /api/bookings/:id/approve
  const approveMatch = pathname.match(/^\/api\/bookings\/([^/]+)\/approve$/);
  if (method === 'POST' && approveMatch) {
    return json(res, 200, { ...BOOKING_PENDING, status: 'payment_pending' });
  }

  // POST /api/bookings/:id/reject
  const rejectMatch = pathname.match(/^\/api\/bookings\/([^/]+)\/reject$/);
  if (method === 'POST' && rejectMatch) {
    return json(res, 200, { ...BOOKING_PENDING, status: 'rejected' });
  }

  // POST /api/bookings/:id/cancel
  const cancelMatch = pathname.match(/^\/api\/bookings\/([^/]+)\/cancel$/);
  if (method === 'POST' && cancelMatch) {
    return json(res, 200, { ...BOOKING_PENDING, status: 'cancelled_by_guest' });
  }

  // GET /api/bookings/:id
  const bookingIdMatch = pathname.match(/^\/api\/bookings\/([^/]+)$/);
  if (method === 'GET' && bookingIdMatch) {
    const id = bookingIdMatch[1];
    const all = [...GUEST_BOOKINGS];
    const booking = all.find(b => b.id === id);
    if (!booking) return notFound(res);
    return json(res, 200, booking);
  }

  // POST /api/bookings  (create)
  if (method === 'POST' && pathname === '/api/bookings') {
    readBody(req).then(body => {
      const listing = ALL_LISTINGS.find(l => l.id === body.listingId) ?? LISTING_TASHKENT;
      const newBooking = {
        ...BOOKING_PENDING,
        id: 'booking-new-001',
        listingId: body.listingId ?? 'listing-001',
        checkIn: body.checkIn ?? BOOKING_PENDING.checkIn,
        checkOut: body.checkOut ?? BOOKING_PENDING.checkOut,
        guests: body.guests ?? 1,
        message: body.message ?? '',
        status: listing.instantBook ? 'payment_pending' : 'pending_host_approval',
      };
      json(res, 201, newBooking);
    });
    return;
  }

  // GET /api/bookings  (guest list)
  if (method === 'GET' && pathname === '/api/bookings') {
    return json(res, 200, { bookings: GUEST_BOOKINGS });
  }

  // ── Payments ──────────────────────────────────────────────────────────────

  if (method === 'POST' && pathname === '/api/payments/checkout') {
    readBody(req).then(() => {
      json(res, 201, CHECKOUT_SESSION);
    });
    return;
  }

  // ── Auth ──────────────────────────────────────────────────────────────────

  if (method === 'POST' && pathname === '/api/auth/login') {
    readBody(req).then(() => {
      // Return 401 — real auth requires mgID OIDC flow
      json(res, 401, { error: 'Invalid credentials' });
    });
    return;
  }

  if (method === 'POST' && pathname === '/api/auth/logout') {
    res.writeHead(200);
    res.end();
    return;
  }

  if (method === 'GET' && pathname === '/api/auth/me') {
    return json(res, 401, { error: 'unauthorized' });
  }

  // ── Admin — Webhook endpoints ─────────────────────────────────────────────

  // DELETE /api/admin/webhooks/endpoints/:id
  const webhookDeleteMatch = pathname.match(/^\/api\/admin\/webhooks\/endpoints\/([^/]+)$/);
  if (method === 'DELETE' && webhookDeleteMatch) {
    res.writeHead(204);
    res.end();
    return;
  }

  // POST /api/admin/webhooks/endpoints
  if (method === 'POST' && pathname === '/api/admin/webhooks/endpoints') {
    readBody(req).then(body => {
      const endpoint = {
        ...WEBHOOK_ENDPOINT_STUB,
        id: 'wh-002',
        url: (body.url as string) ?? 'https://hooks.example.com/new',
        description: body.description as string | undefined,
        eventTypes: Array.isArray(body.eventTypes)
          ? body.eventTypes as string[]
          : WEBHOOK_ENDPOINT_STUB.eventTypes,
      };
      json(res, 201, endpoint);
    });
    return;
  }

  // GET /api/admin/webhooks/endpoints
  if (method === 'GET' && pathname === '/api/admin/webhooks/endpoints') {
    return json(res, 200, { endpoints: [WEBHOOK_ENDPOINT_STUB] });
  }

  // ── Default ───────────────────────────────────────────────────────────────

  notFound(res);
}

// ---------------------------------------------------------------------------
// Server lifecycle
// ---------------------------------------------------------------------------

export interface MockGateway {
  port: number;
  close(): Promise<void>;
}

export async function startMockGateway(port = 9999): Promise<MockGateway> {
  const server = http.createServer(handle);

  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, '127.0.0.1', resolve);
  });

  return {
    port,
    close: () =>
      new Promise<void>((resolve, reject) =>
        server.close(err => (err ? reject(err) : resolve()))
      ),
  };
}

// ---------------------------------------------------------------------------
// Standalone: tsx e2e/mock-gateway.ts
// ---------------------------------------------------------------------------

import { fileURLToPath } from 'url';
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  startMockGateway(9999).then(gw => {
    console.log(`[mock-gateway] running on http://localhost:${gw.port} — Ctrl+C to stop`);
  });
}
