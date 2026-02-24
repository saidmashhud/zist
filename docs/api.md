# Zist API Reference

All API routes are accessible through the Gateway at `:8000`. The `/api` prefix is stripped before forwarding to upstream services.

## Gateway

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/healthz` | none | Health check |
| GET | `/api/auth/login` | none | Initiate OIDC PKCE login → 302 redirect to mgID |
| GET | `/api/auth/callback` | none | OAuth2 callback → exchanges code for JWT, sets cookie |
| POST | `/api/auth/logout` | none | Clears `zist_session` cookie |
| GET | `/api/auth/me` | cookie | Returns authenticated user info from JWT claims |
| GET | `/api/admin/webhooks` | `zist.webhooks.manage` | List webhook endpoints (proxied to mgEvents) |
| POST | `/api/admin/webhooks` | `zist.webhooks.manage` | Create webhook endpoint |
| GET | `/api/admin/webhooks/:id/deliveries` | `zist.webhooks.manage` | List deliveries |
| POST | `/api/admin/webhooks/:id/rotate-secret` | `zist.webhooks.manage` | Rotate signing secret |
| DELETE | `/api/admin/webhooks/:id` | `zist.webhooks.manage` | Delete endpoint |
| POST | `/api/admin/webhooks/:id/deliveries/:did/retry` | `zist.webhooks.manage` | Retry delivery |

## Listings Service

Base URL: `/api/listings` (via gateway) or `:8001/listings` (direct)

### List Listings

```
GET /listings
```

Public. Returns up to 50 listings ordered by creation time.

**Response 200:**
```json
{
  "listings": [
    {
      "id": "uuid",
      "title": "Cozy Apartment in Tashkent",
      "description": "...",
      "city": "Tashkent",
      "country": "UZ",
      "pricePerNight": "250000.00",
      "currency": "UZS",
      "maxGuests": 4,
      "hostId": "user-uuid",
      "createdAt": 1740000000,
      "updatedAt": 1740000000
    }
  ]
}
```

### Get Listing

```
GET /listings/:id
```

Public.

**Response 200:** Single listing object.
**Response 404:** `{"error": "listing not found"}`

### Create Listing

```
POST /listings
```

Auth: `zist.listings.manage`

**Request:**
```json
{
  "title": "Cozy Apartment in Tashkent",
  "description": "A beautiful place to stay",
  "city": "Tashkent",
  "country": "UZ",
  "pricePerNight": "250000.00",
  "currency": "UZS",
  "maxGuests": 4,
  "hostId": "user-uuid"
}
```

**Response 201:** Created listing with generated `id`.
**Response 401:** `{"error": "unauthorized"}`
**Response 403:** `{"error": "insufficient_scope", "required": "zist.listings.manage"}`
**Response 422:** `{"error": "title, city, and pricePerNight are required"}`

### Update Listing

```
PUT /listings/:id
```

Auth: `zist.listings.manage`

**Request:**
```json
{
  "title": "Updated Title",
  "description": "Updated description",
  "pricePerNight": "300000.00",
  "maxGuests": 6
}
```

**Response 204:** No content.
**Response 404:** Listing not found.

### Delete Listing

```
DELETE /listings/:id
```

Auth: `zist.listings.manage`

**Response 204:** No content.
**Response 404:** Listing not found.

---

## Bookings Service

Base URL: `/api/bookings` (via gateway) or `:8002/bookings` (direct)

### List Bookings

```
GET /bookings
```

Auth: `zist.bookings.read`. Returns only the authenticated user's bookings.

**Response 200:**
```json
{
  "bookings": [
    {
      "id": "uuid",
      "listingId": "listing-uuid",
      "guestId": "user-uuid",
      "checkIn": "2026-04-01",
      "checkOut": "2026-04-05",
      "guests": 2,
      "totalAmount": "1000000.00",
      "currency": "UZS",
      "status": "pending",
      "checkoutId": null,
      "createdAt": 1740000000,
      "updatedAt": 1740000000
    }
  ]
}
```

### Get Booking

```
GET /bookings/:id
```

Public (no auth required).

### Create Booking

```
POST /bookings
```

Auth: `zist.bookings.manage`. Guest ID is set from the authenticated principal.

**Request:**
```json
{
  "listingId": "listing-uuid",
  "checkIn": "2026-04-01",
  "checkOut": "2026-04-05",
  "guests": 2,
  "totalAmount": "1000000.00",
  "currency": "UZS"
}
```

**Response 201:** Created booking with `status: "pending"`.

### Confirm Booking (internal)

```
POST /bookings/:id/confirm
```

Auth: `X-Internal-Token` header required. Transitions `pending` → `confirmed`.

**Response 204:** No content.
**Response 403:** `{"error": "forbidden", "code": "INVALID_INTERNAL_TOKEN"}`

### Fail Booking (internal)

```
POST /bookings/:id/fail
```

Auth: `X-Internal-Token`. Transitions `pending` → `failed`.

### Cancel Booking (internal)

```
POST /bookings/:id/cancel
```

Auth: `X-Internal-Token`. Cancels any non-cancelled booking.

### Set Checkout ID (internal)

```
PUT /bookings/:id/checkout
```

Auth: `X-Internal-Token`.

**Request:**
```json
{"checkoutId": "session-uuid"}
```

---

## Payments Service

Base URL: `/api/payments` (via gateway) or `:8003` (direct)

### Create Checkout

```
POST /checkout
```

Auth: `zist.payments.create`

**Request:**
```json
{
  "listingId": "listing-uuid",
  "bookingId": "booking-uuid",
  "amount": "500000.00",
  "currency": "UZS",
  "successUrl": "http://localhost:3000/bookings/{id}/success",
  "cancelUrl": "http://localhost:3000/bookings/{id}/cancel",
  "customerEmail": "guest@example.com"
}
```

**Response 201:**
```json
{
  "sessionId": "checkout-session-uuid",
  "checkoutUrl": "https://checkout.mashgate.local/session/..."
}
```

**Response 401:** Unauthorized.
**Response 403:** Insufficient scope.
**Response 502:** Mashgate unavailable.

### Receive Mashgate Webhook

```
POST /webhooks/mashgate
```

Auth: none (signature-verified internally). Processes payment events and triggers booking status transitions.

**Handled events:**
- `payment.captured` → confirms booking
- `payment.failed` / `payment_capture.failed` → fails booking
- `checkout.completed` → logged
- `checkout.expired` → logged

**Response 200:** `{"status": "ok"}` (new event) or `{"status": "ok", "dedup": "skipped"}` (duplicate)

---

## Error Codes

| HTTP Status | Meaning |
|-------------|---------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (success, no body) |
| 400 | Bad Request (invalid JSON) |
| 401 | Unauthorized (no auth or invalid session) |
| 403 | Forbidden (missing scope or invalid internal token) |
| 404 | Not Found |
| 422 | Unprocessable Entity (missing required fields) |
| 502 | Bad Gateway (upstream service unavailable) |
