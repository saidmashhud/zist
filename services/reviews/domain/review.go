// Package domain defines the Review entity and related types.
package domain

// Review represents a guest's review of a completed stay.
type Review struct {
	ID        string  `json:"id"`
	BookingID string  `json:"bookingId"`
	ListingID string  `json:"listingId"`
	GuestID   string  `json:"guestId"`
	HostID    string  `json:"hostId"`
	TenantID  string  `json:"tenantId"`
	Rating    int     `json:"rating"`   // 1–5
	Comment   string  `json:"comment"`
	Reply     string  `json:"reply,omitempty"` // host reply
	CreatedAt int64   `json:"createdAt"`
	UpdatedAt int64   `json:"updatedAt"`
}

// CreateReviewInput holds the fields required to create a review.
type CreateReviewInput struct {
	BookingID string
	ListingID string
	GuestID   string
	HostID    string
	TenantID  string
	Rating    int
	Comment   string
}
