// Package domain defines the core domain types for the listings service.
package domain

// Listing represents a rental property listing.
type Listing struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// Location
	City    string `json:"city"`
	Country string `json:"country"`
	Address string `json:"address"`
	// Property
	Type      string `json:"type"` // apartment|house|guesthouse|room
	Bedrooms  int    `json:"bedrooms"`
	Beds      int    `json:"beds"`
	Bathrooms int    `json:"bathrooms"`
	MaxGuests int    `json:"maxGuests"`
	// Amenities & Rules
	Amenities []string   `json:"amenities"`
	Rules     HouseRules `json:"rules"`
	// Pricing
	PricePerNight string `json:"pricePerNight"` // decimal string e.g. "150000.00"
	Currency      string `json:"currency"`
	CleaningFee   string `json:"cleaningFee"`
	Deposit       string `json:"deposit"`
	// Stay constraints
	MinNights int `json:"minNights"`
	MaxNights int `json:"maxNights"`
	// Booking settings
	CancellationPolicy string `json:"cancellationPolicy"` // flexible|moderate|strict
	InstantBook        bool   `json:"instantBook"`
	// Status & ratings
	Status        string  `json:"status"` // draft|active|paused
	AverageRating float64 `json:"averageRating"`
	ReviewCount   int     `json:"reviewCount"`
	// Meta
	HostID    string `json:"hostId"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	// Computed (loaded separately)
	Photos []Photo `json:"photos,omitempty"`
}

// HouseRules describes behaviour rules for a listing.
type HouseRules struct {
	CheckInFrom    string `json:"checkInFrom"`
	CheckOutBefore string `json:"checkOutBefore"`
	QuietHoursFrom string `json:"quietHoursFrom"`
	QuietHoursTo   string `json:"quietHoursTo"`
	Smoking        bool   `json:"smoking"`
	Pets           bool   `json:"pets"`
	Parties        bool   `json:"parties"`
}

// Photo is an ordered image attached to a listing.
type Photo struct {
	ID        string `json:"id"`
	ListingID string `json:"listingId"`
	URL       string `json:"url"`
	Caption   string `json:"caption"`
	SortOrder int    `json:"sortOrder"`
	CreatedAt int64  `json:"createdAt"`
}

// AvailabilityDay is a single calendar day entry for a listing.
type AvailabilityDay struct {
	Date          string `json:"date"`   // YYYY-MM-DD
	Status        string `json:"status"` // available|blocked|booked
	PriceOverride string `json:"priceOverride,omitempty"`
	BookingID     string `json:"bookingId,omitempty"`
}

// PricePreview is the full cost breakdown returned before booking.
type PricePreview struct {
	Nights           int    `json:"nights"`
	PricePerNight    string `json:"pricePerNight"`
	Subtotal         string `json:"subtotal"`
	CleaningFee      string `json:"cleaningFee"`
	PlatformFeeGuest string `json:"platformFeeGuest"`
	Total            string `json:"total"`
	Currency         string `json:"currency"`
}

// CreateListingInput holds validated fields for a new listing.
type CreateListingInput struct {
	TenantID           string
	HostID             string
	Title              string
	Description        string
	City               string
	Country            string
	Address            string
	Type               string
	Bedrooms           int
	Beds               int
	Bathrooms          int
	MaxGuests          int
	Amenities          []string
	Rules              HouseRules
	PricePerNight      string
	Currency           string
	CleaningFee        string
	Deposit            string
	MinNights          int
	MaxNights          int
	CancellationPolicy string
	InstantBook        bool
}

// UpdateListingInput holds optional fields for a partial update.
type UpdateListingInput struct {
	Title              *string
	Description        *string
	Address            *string
	Type               *string
	Bedrooms           *int
	Beds               *int
	Bathrooms          *int
	MaxGuests          *int
	Amenities          []string
	Rules              *HouseRules
	PricePerNight      *string
	Currency           *string
	CleaningFee        *string
	Deposit            *string
	MinNights          *int
	MaxNights          *int
	CancellationPolicy *string
	InstantBook        *bool
	Status             *string
}

// SearchFilters holds all query parameters for listing search.
type SearchFilters struct {
	City            string
	CheckIn         string
	CheckOut        string
	Guests          int
	Type            string
	MinPrice        string
	MaxPrice        string
	Amenities       []string
	InstantBookOnly bool
	Limit           int
}
