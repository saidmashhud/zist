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
  LISTING_SAMARKAND,
  GUEST_BOOKINGS,
  HOST_BOOKINGS,
  BOOKING_PENDING,
  BOOKING_CONFIRMED,
  BOOKING_CANCELLED,
  PRICE_PREVIEW_3N,
  PRICE_PREVIEW_4N,
  CHECKOUT_SESSION,
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

function ok(res: ServerResponse): void {
  res.writeHead(200);
  res.end();
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
    res.writeHead(204, { 'Access-Control-Allow-Origin': '*', 'Access-Control-Allow-Methods': 'GET,POST,PATCH,DELETE' });
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
    // Simple night count for preview selection
    const nights = checkIn && checkOut
      ? Math.max(1, Math.round((new Date(checkOut).getTime() - new Date(checkIn).getTime()) / 86_400_000))
      : 3;
    const preview = nights >= 4 ? PRICE_PREVIEW_4N : PRICE_PREVIEW_3N;
    return json(res, 200, { ...preview, nights });
  }

  // PATCH /api/listings/:id
  const listingIdMatch = pathname.match(/^\/api\/listings\/([^/]+)$/);
  if (method === 'PATCH' && listingIdMatch) {
    const id = listingIdMatch[1];
    const listing = ALL_LISTINGS.find(l => l.id === id);
    if (!listing) return notFound(res);
    return json(res, 200, { ...listing, status: 'paused' });
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
    let body = '';
    req.on('data', chunk => { body += chunk; });
    req.on('end', () => {
      const input = JSON.parse(body || '{}');
      const listing = ALL_LISTINGS.find(l => l.id === input.listingId) ?? LISTING_TASHKENT;
      const newBooking = {
        ...BOOKING_PENDING,
        id: 'booking-new-001',
        listingId: input.listingId ?? 'listing-001',
        checkIn: input.checkIn ?? BOOKING_PENDING.checkIn,
        checkOut: input.checkOut ?? BOOKING_PENDING.checkOut,
        guests: input.guests ?? 1,
        message: input.message ?? '',
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
    let body = '';
    req.on('data', chunk => { body += chunk; });
    req.on('end', () => {
      json(res, 201, CHECKOUT_SESSION);
    });
    return;
  }

  // ── Auth ──────────────────────────────────────────────────────────────────

  if (method === 'POST' && pathname === '/api/auth/logout') {
    res.writeHead(200);
    res.end();
    return;
  }

  if (method === 'GET' && pathname === '/api/auth/me') {
    // Return 401 — auth is cookie-based, not bearer
    return json(res, 401, { error: 'unauthorized' });
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
