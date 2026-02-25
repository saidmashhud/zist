package domain

// SearchFilters are the parameters accepted by the search endpoint.
type SearchFilters struct {
	City            string
	Lat             float64
	Lng             float64
	RadiusKM        float64
	CheckIn         string // YYYY-MM-DD
	CheckOut        string // YYYY-MM-DD
	Guests          int
	Type            string
	MinPrice        string
	MaxPrice        string
	Amenities       []string
	InstantBookOnly bool
	SortBy          string // rating, price, distance (default: rating)
	Limit           int
	Offset          int
}

// SearchResult is a single listing returned from a search query.
type SearchResult struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	City          string   `json:"city"`
	Country       string   `json:"country"`
	Type          string   `json:"type"`
	PricePerNight string   `json:"pricePerNight"`
	Currency      string   `json:"currency"`
	MaxGuests     int      `json:"maxGuests"`
	InstantBook   bool     `json:"instantBook"`
	AverageRating float64  `json:"averageRating"`
	ReviewCount   int      `json:"reviewCount"`
	CoverPhoto    string   `json:"coverPhoto,omitempty"`
	Amenities     []string `json:"amenities"`
	DistanceKM    *float64 `json:"distanceKm,omitempty"`
}

// SearchResponse wraps search results with pagination metadata.
type SearchResponse struct {
	Listings []SearchResult `json:"listings"`
	Total    int            `json:"total"`
	Limit    int            `json:"limit"`
	Offset   int            `json:"offset"`
}
