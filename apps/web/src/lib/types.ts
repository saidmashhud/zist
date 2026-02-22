export interface Listing {
  id: string;
  title: string;
  description: string;
  city: string;
  country: string;
  pricePerNight: string;
  currency: string;
  maxGuests: number;
  hostId: string;
  createdAt: number;
  updatedAt: number;
}

export interface Booking {
  id: string;
  listingId: string;
  guestId: string;
  checkIn: string;
  checkOut: string;
  guests: number;
  totalAmount: string;
  currency: string;
  status: 'pending' | 'confirmed' | 'cancelled';
  createdAt: number;
  updatedAt: number;
}

export interface CheckoutResult {
  sessionId: string;
  checkoutUrl: string;
}
