export default {
  // Navigation
  nav: {
    home: 'Home',
    search: 'Search',
    bookings: 'Bookings',
    host: 'Host',
    login: 'Sign in',
    logout: 'Sign out',
  },
  // Home page
  home: {
    hero_title: 'Find your perfect stay in Central Asia',
    hero_subtitle: 'Apartments, houses, and guesthouses across Uzbekistan, Kazakhstan, and beyond.',
    search_placeholder: 'City, region, or landmark',
    search_btn: 'Search',
    guests_label: 'Guests',
  },
  // Listing
  listing: {
    book_now: 'Book now',
    instant_book: 'Instant booking',
    request_book: 'Request to book',
    per_night: '/ night',
    guests: 'guests',
    nights: 'nights',
    cleaning_fee: 'Cleaning fee',
    platform_fee: 'Platform fee',
    total: 'Total',
    check_in: 'Check-in',
    check_out: 'Check-out',
    reviews: 'Reviews',
    no_reviews: 'No reviews yet.',
    amenities: 'Amenities',
    house_rules: 'House rules',
    cancellation: 'Cancellation policy',
    flexible: 'Flexible',
    moderate: 'Moderate',
    strict: 'Strict',
  },
  // Booking
  booking: {
    status_pending_host_approval: 'Awaiting host approval',
    status_payment_pending: 'Payment pending',
    status_confirmed: 'Confirmed',
    status_cancelled_by_guest: 'Cancelled by you',
    status_cancelled_by_host: 'Cancelled by host',
    status_rejected: 'Rejected',
    status_failed: 'Payment failed',
    status_completed: 'Completed',
    cancel: 'Cancel booking',
    approve: 'Approve',
    reject: 'Reject',
  },
  // Reviews
  reviews: {
    write_review: 'Write a review',
    your_rating: 'Your rating',
    your_comment: 'Your experience',
    submit: 'Submit review',
    reply: 'Reply',
    host_reply: 'Host reply',
  },
  // Host
  host: {
    dashboard: 'Host dashboard',
    my_listings: 'My listings',
    my_bookings: 'Bookings on my listings',
    new_listing: 'Create listing',
    edit_listing: 'Edit listing',
    publish: 'Publish',
    unpublish: 'Unpublish',
    instant_book_label: 'Allow instant booking',
  },
  // Errors
  errors: {
    generic: 'Something went wrong. Please try again.',
    not_found: 'Not found.',
    unauthorized: 'Please sign in to continue.',
    dates_unavailable: 'Selected dates are not available.',
  },
  // Currency
  currency: {
    UZS: 'UZS',
    USD: 'USD',
    KZT: 'KZT',
  },
} as const;
