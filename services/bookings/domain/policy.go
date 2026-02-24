package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// CalculateRefund returns the refund amount based on cancellation policy and time until check-in.
//
// Policies:
//
//	flexible:  ≥ 24h before check-in → 100%  |  < 24h → 0%
//	moderate:  ≥ 5 days → 100%  |  1–4 days (≥ 24h) → 50%  |  < 24h → 0%
//	strict:    ≥ 14 days → 50%  |  < 14 days → 0%
func CalculateRefund(policy, totalAmount, currency, checkIn string) (RefundResult, error) {
	checkInDate, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return RefundResult{}, fmt.Errorf("invalid check_in date: %w", err)
	}

	hoursUntil := time.Until(checkInDate).Hours()
	daysUntil := hoursUntil / 24.0

	total, err := strconv.ParseFloat(strings.TrimSpace(totalAmount), 64)
	if err != nil {
		return RefundResult{}, fmt.Errorf("invalid total_amount: %w", err)
	}

	var pct int
	switch policy {
	case "flexible":
		if hoursUntil >= 24 {
			pct = 100
		}
	case "moderate":
		if daysUntil >= 5 {
			pct = 100
		} else if hoursUntil >= 24 {
			pct = 50
		}
	case "strict":
		if daysUntil >= 14 {
			pct = 50
		}
	default:
		pct = 0
	}

	refund := math.Round(total*float64(pct)) / 100.0
	return RefundResult{
		RefundAmount: fmt.Sprintf("%.2f", refund),
		RefundPct:    pct,
		Currency:     currency,
	}, nil
}
