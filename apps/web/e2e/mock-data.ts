/**
 * Shared test fixture data for the mock gateway.
 * All monetary values are in UZS (Uzbekistani Som).
 */

export const LISTING_TASHKENT = {
  id: 'listing-001',
  hostId: 'host-001',
  title: 'Cozy Apartment in Tashkent',
  type: 'apartment',
  city: 'Tashkent',
  country: 'Uzbekistan',
  address: 'Amir Temur St 1',
  description: 'A charming apartment in the heart of Tashkent. Close to all major attractions.',
  pricePerNight: '150000',
  currency: 'UZS',
  status: 'active',
  instantBook: false,
  maxGuests: 4,
  bedrooms: 2,
  beds: 2,
  bathrooms: 1,
  minNights: 1,
  amenities: ['wifi', 'kitchen', 'ac'],
  cancellationPolicy: 'flexible',
  rules: {
    checkInFrom: '14:00',
    checkOutBefore: '12:00',
    quietHoursFrom: '22:00',
    quietHoursTo: '08:00',
    smoking: false,
    pets: true,
    parties: false,
  },
  photos: [],
  averageRating: 4.8,
  reviewCount: 12,
};

export const LISTING_SAMARKAND = {
  id: 'listing-002',
  hostId: 'host-001',
  title: 'Samarkand Historical Suite',
  type: 'house',
  city: 'Samarkand',
  country: 'Uzbekistan',
  address: 'Registan Square 5',
  description: 'Stunning house next to Registan. Wake up to centuries of history.',
  pricePerNight: '200000',
  currency: 'UZS',
  status: 'active',
  instantBook: true,
  maxGuests: 6,
  bedrooms: 3,
  beds: 4,
  bathrooms: 2,
  minNights: 2,
  amenities: ['wifi', 'kitchen', 'ac', 'parking'],
  cancellationPolicy: 'moderate',
  rules: null,
  photos: [],
  averageRating: 0,
  reviewCount: 0,
};

export const LISTING_PAUSED = {
  id: 'listing-003',
  hostId: 'host-001',
  title: 'Bukhara Silk Road Retreat',
  type: 'guesthouse',
  city: 'Bukhara',
  country: 'Uzbekistan',
  address: 'Old Town 12',
  description: 'Traditional guesthouse in the old city.',
  pricePerNight: '120000',
  currency: 'UZS',
  status: 'paused',
  instantBook: false,
  maxGuests: 2,
  bedrooms: 1,
  beds: 1,
  bathrooms: 1,
  minNights: 1,
  amenities: ['wifi'],
  cancellationPolicy: 'strict',
  rules: null,
  photos: [],
  averageRating: 4.5,
  reviewCount: 3,
};

export const ALL_LISTINGS = [LISTING_TASHKENT, LISTING_SAMARKAND, LISTING_PAUSED];
export const HOST_OWN_LISTINGS = [LISTING_TASHKENT, LISTING_SAMARKAND, LISTING_PAUSED];

export const BOOKING_PENDING = {
  id: 'booking-001',
  listingId: 'listing-001',
  guestId: 'test-user-001',
  hostId: 'host-001',
  checkIn: '2026-03-10',
  checkOut: '2026-03-13',
  guests: 2,
  status: 'pending_host_approval',
  totalAmount: '513000',
  platformFee: '33000',
  cleaningFee: '30000',
  currency: 'UZS',
  cancellationPolicy: 'flexible',
  message: 'Looking forward to the stay!',
  paymentId: '',
  checkoutId: '',
  createdAt: '2026-02-20T10:00:00Z',
  updatedAt: '2026-02-20T10:00:00Z',
};

export const BOOKING_CONFIRMED = {
  id: 'booking-002',
  listingId: 'listing-002',
  guestId: 'test-user-001',
  hostId: 'host-001',
  checkIn: '2026-04-01',
  checkOut: '2026-04-05',
  guests: 4,
  status: 'confirmed',
  totalAmount: '872000',
  platformFee: '56000',
  cleaningFee: '40000',
  currency: 'UZS',
  cancellationPolicy: 'moderate',
  message: '',
  paymentId: 'pay-abc123',
  checkoutId: '',
  createdAt: '2026-02-15T08:00:00Z',
  updatedAt: '2026-02-16T10:00:00Z',
};

export const BOOKING_CANCELLED = {
  id: 'booking-003',
  listingId: 'listing-001',
  guestId: 'test-user-001',
  hostId: 'host-001',
  checkIn: '2026-01-10',
  checkOut: '2026-01-13',
  guests: 1,
  status: 'cancelled_by_guest',
  totalAmount: '480000',
  platformFee: '30000',
  cleaningFee: '25000',
  currency: 'UZS',
  cancellationPolicy: 'flexible',
  message: '',
  paymentId: '',
  checkoutId: '',
  createdAt: '2025-12-20T10:00:00Z',
  updatedAt: '2025-12-22T08:00:00Z',
};

export const GUEST_BOOKINGS = [BOOKING_PENDING, BOOKING_CONFIRMED, BOOKING_CANCELLED];
export const HOST_BOOKINGS = [BOOKING_PENDING, BOOKING_CONFIRMED];

export const PRICE_PREVIEW_3N = {
  nights: 3,
  pricePerNight: '150000',
  subtotal: '450000',
  cleaningFee: '30000',
  platformFeeGuest: '33000',
  total: '513000',
  currency: 'UZS',
};

export const PRICE_PREVIEW_4N = {
  nights: 4,
  pricePerNight: '200000',
  subtotal: '800000',
  cleaningFee: '40000',
  platformFeeGuest: '56000',
  total: '896000',
  currency: 'UZS',
};

export const CHECKOUT_SESSION = {
  sessionId: 'sess-mock-001',
  checkoutUrl: 'https://pay.mashgate.test/checkout/sess-mock-001',
};

/** Fake JWT payload for the test guest user. */
export const MOCK_GUEST = {
  user_id: 'test-user-001',
  email: 'test@zist.test',
  tenant_id: 'tenant-001',
  scope:
    'zist.listings.read zist.listings.manage zist.bookings.read zist.bookings.manage zist.payments.create zist.webhooks.manage',
  exp: 9_999_999_999,
};

/** Fake JWT payload for the test host user. */
export const MOCK_HOST = {
  user_id: 'host-001',
  email: 'host@zist.test',
  tenant_id: 'tenant-001',
  scope:
    'zist.listings.read zist.listings.manage zist.bookings.read zist.bookings.manage',
  exp: 9_999_999_999,
};
