package models

import "time"

// CouponDiscountType is the business category (PRD section 6.4) — it
// controls WHICH items a coupon's discount applies to and what extra
// matching rule (if any) is required.
type CouponDiscountType string

const (
	CouponDiscountTypeTotalBelanja CouponDiscountType = "total_belanja"
	CouponDiscountTypeItemTertentu CouponDiscountType = "item_tertentu"
	CouponDiscountTypeEvent        CouponDiscountType = "event"
	CouponDiscountTypeBundle       CouponDiscountType = "bundle"
)

var ValidCouponDiscountTypes = map[CouponDiscountType]bool{
	CouponDiscountTypeTotalBelanja: true,
	CouponDiscountTypeItemTertentu: true,
	CouponDiscountTypeEvent:        true,
	CouponDiscountTypeBundle:       true,
}

// CouponDiscountValueType controls HOW discount_value is applied — a flat
// Rupiah amount or a percentage. Orthogonal to CouponDiscountType.
type CouponDiscountValueType string

const (
	CouponDiscountValueTypeFixed      CouponDiscountValueType = "fixed"
	CouponDiscountValueTypePercentage CouponDiscountValueType = "percentage"
)

var ValidCouponDiscountValueTypes = map[CouponDiscountValueType]bool{
	CouponDiscountValueTypeFixed:      true,
	CouponDiscountValueTypePercentage: true,
}

type Coupon struct {
	ID                  string                  `json:"id"`
	Code                string                  `json:"code"`
	DiscountType        CouponDiscountType      `json:"discount_type"`
	DiscountValueType   CouponDiscountValueType `json:"discount_value_type"`
	DiscountValue       float64                 `json:"discount_value"`
	MinSpend            int64                   `json:"min_spend"`
	ValidFrom           time.Time               `json:"valid_from"`
	ValidUntil          time.Time               `json:"valid_until"`
	MaxTotalUsage       *int32                  `json:"max_total_usage"`
	MaxUsagePerCustomer *int32                  `json:"max_usage_per_customer"`
	CurrentUsageCount   int32                   `json:"current_usage_count"`
	IsActive            bool                    `json:"is_active"`
	ProductIDs          []string                `json:"product_ids,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
}
