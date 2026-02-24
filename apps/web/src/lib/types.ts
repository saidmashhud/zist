// ─── Listings ────────────────────────────────────────────────────────────────

export type PropertyType = 'apartment' | 'house' | 'guesthouse' | 'room';
export type CancellationPolicy = 'flexible' | 'moderate' | 'strict';
export type ListingStatus = 'draft' | 'active' | 'paused';
export type AvailabilityStatus = 'available' | 'blocked' | 'booked';

export interface HouseRules {
  checkInFrom?: string;    // "14:00"
  checkOutBefore?: string; // "12:00"
  quietHoursFrom?: string;
  quietHoursTo?: string;
  smoking: boolean;
  pets: boolean;
  parties: boolean;
}

export interface Photo {
  id: string;
  listingId: string;
  url: string;
  caption: string;
  sortOrder: number;
  createdAt: number;
}

export interface Listing {
  id: string;
  title: string;
  description: string;
  // Location
  city: string;
  country: string;
  address: string;
  // Property
  type: PropertyType;
  bedrooms: number;
  beds: number;
  bathrooms: number;
  maxGuests: number;
  // Amenities & Rules
  amenities: string[];
  rules: HouseRules;
  // Pricing
  pricePerNight: string;
  currency: string;
  cleaningFee: string;
  deposit: string;
  // Stay constraints
  minNights: number;
  maxNights: number;
  // Booking settings
  cancellationPolicy: CancellationPolicy;
  instantBook: boolean;
  // Status & ratings
  status: ListingStatus;
  averageRating: number;
  reviewCount: number;
  // Meta
  hostId: string;
  createdAt: number;
  updatedAt: number;
  // Computed
  photos?: Photo[];
}

export interface AvailabilityDay {
  date: string; // YYYY-MM-DD
  status: AvailabilityStatus;
  priceOverride?: string;
  bookingId?: string;
}

export interface PricePreview {
  nights: number;
  pricePerNight: string;
  subtotal: string;
  cleaningFee: string;
  platformFeeGuest: string;
  total: string;
  currency: string;
}

export interface SearchFilters {
  city?: string;
  checkIn?: string;  // YYYY-MM-DD
  checkOut?: string; // YYYY-MM-DD
  guests?: number;
  type?: PropertyType;
  minPrice?: string;
  maxPrice?: string;
  amenities?: string[];
  instantBook?: boolean;
}

// ─── Bookings ────────────────────────────────────────────────────────────────

export type BookingStatus =
  | 'pending_host_approval'
  | 'payment_pending'
  | 'confirmed'
  | 'cancelled_by_guest'
  | 'cancelled_by_host'
  | 'rejected'
  | 'failed'
  | 'completed';

export interface Booking {
  id: string;
  listingId: string;
  guestId: string;
  checkIn: string;  // YYYY-MM-DD
  checkOut: string; // YYYY-MM-DD
  guests: number;
  totalAmount: string;
  platformFee: string;
  cleaningFee: string;
  currency: string;
  status: BookingStatus;
  cancellationPolicy: CancellationPolicy;
  message?: string;
  checkoutId?: string;
  approvedAt?: number;
  expiresAt?: number;
  createdAt: number;
  updatedAt: number;
}

// ─── Payments ────────────────────────────────────────────────────────────────

export interface CheckoutResult {
  sessionId: string;
  checkoutUrl: string;
}

// ─── Auth ────────────────────────────────────────────────────────────────────

export interface User {
  sub: string;
  email: string;
  name?: string;
  tenant_id: string;
  roles: string[];
  scope: string;
}

// ─── Common amenity codes (for UI dropdowns) ─────────────────────────────────

export const AMENITIES = [
  { code: 'wifi',      label: 'Wi-Fi' },
  { code: 'kitchen',   label: 'Kitchen' },
  { code: 'parking',   label: 'Free parking' },
  { code: 'pool',      label: 'Swimming pool' },
  { code: 'gym',       label: 'Gym' },
  { code: 'ac',        label: 'Air conditioning' },
  { code: 'heating',   label: 'Heating' },
  { code: 'washer',    label: 'Washer' },
  { code: 'dryer',     label: 'Dryer' },
  { code: 'tv',        label: 'TV' },
  { code: 'workspace', label: 'Dedicated workspace' },
  { code: 'balcony',   label: 'Balcony' },
  { code: 'bbq',       label: 'BBQ grill' },
  { code: 'ev_charger','label': 'EV charger' },
] as const;

export type AmenityCode = typeof AMENITIES[number]['code'];
