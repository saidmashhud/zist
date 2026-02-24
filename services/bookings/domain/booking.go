// Package domain defines the core domain types for the bookings service.
package domain

// Booking represents a reservation on a listing.
type Booking struct {
	ID                 string  `json:"id"`
	ListingID          string  `json:"listingId"`
	GuestID            string  `json:"guestId"`
	HostID             string  `json:"hostId"`
	CheckIn            string  `json:"checkIn"`
	CheckOut           string  `json:"checkOut"`
	Guests             int     `json:"guests"`
	TotalAmount        string  `json:"totalAmount"`
	PlatformFee        string  `json:"platformFee"`
	CleaningFee        string  `json:"cleaningFee"`
	Currency           string  `json:"currency"`
	Status             string  `json:"status"`
	CancellationPolicy string  `json:"cancellationPolicy"`
	Message            string  `json:"message,omitempty"`
	CheckoutID         *string `json:"checkoutId,omitempty"`
	ApprovedAt         *int64  `json:"approvedAt,omitempty"`
	ExpiresAt          *int64  `json:"expiresAt,omitempty"`
	PaymentID          *string `json:"paymentId,omitempty"`
	CreatedAt          int64   `json:"createdAt"`
	UpdatedAt          int64   `json:"updatedAt"`
}

// Booking status constants — the full lifecycle state machine.
const (
	StatusPendingHostApproval = "pending_host_approval"
	StatusPaymentPending      = "payment_pending"
	StatusConfirmed           = "confirmed"
	StatusCancelledByGuest    = "cancelled_by_guest"
	StatusCancelledByHost     = "cancelled_by_host"
	StatusRejected            = "rejected"
	StatusFailed              = "failed"
	StatusCompleted           = "completed"
)

// ListingInfo holds the fields fetched from the listings service at booking creation time.
type ListingInfo struct {
	ID                 string
	HostID             string
	InstantBook        bool
	CancellationPolicy string
	PricePerNight      string
	CleaningFee        string
	Currency           string
	MinNights          int
	MaxNights          int
	MaxGuests          int
	Status             string
}

// RefundResult holds the calculated refund amount for a cancellation.
type RefundResult struct {
	RefundAmount string `json:"refundAmount"`
	RefundPct    int    `json:"refundPct"` // 0, 50, or 100
	Currency     string `json:"currency"`
}
