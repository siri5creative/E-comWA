package models

import "time"

type OrderChannel string

const (
	OrderChannelOnline OrderChannel = "online"
	OrderChannelPOS    OrderChannel = "pos"
)

type OrderStatus string

const (
	OrderStatusMenungguKonfirmasi OrderStatus = "menunggu_konfirmasi"
	OrderStatusMenungguPembayaran OrderStatus = "menunggu_pembayaran"
	OrderStatusDiproses           OrderStatus = "diproses"
	OrderStatusDikirim            OrderStatus = "dikirim"
	OrderStatusSelesai            OrderStatus = "selesai"
	OrderStatusDibatalkan         OrderStatus = "dibatalkan"
)

// Order mirrors the orders table (PRD section 7). ShippingCost is not set
// at checkout time — it stays 0 until an admin fills it in later (PRD
// section 6.3).
type Order struct {
	ID             string       `json:"order_id"`
	InvoiceNumber  string       `json:"invoice_number"`
	Channel        OrderChannel `json:"channel"`
	Status         OrderStatus  `json:"status"`
	CouponCode     *string      `json:"coupon_code,omitempty"`
	Subtotal       int64        `json:"subtotal"`
	DiscountAmount int64        `json:"discount_amount"`
	ShippingCost   int64        `json:"shipping_cost"`
	Total          int64        `json:"total"`
	CreatedAt      time.Time    `json:"created_at"`
}

type OrderItem struct {
	ProductVariantID string `json:"product_variant_id"`
	Quantity         int32  `json:"quantity"`
	PriceAtPurchase  int64  `json:"price_at_purchase"`
}

// ValidOrderStatuses is the full set of allowed order_status enum values.
var ValidOrderStatuses = map[OrderStatus]bool{
	OrderStatusMenungguKonfirmasi: true,
	OrderStatusMenungguPembayaran: true,
	OrderStatusDiproses:           true,
	OrderStatusDikirim:            true,
	OrderStatusSelesai:            true,
	OrderStatusDibatalkan:         true,
}

// AllowedOrderStatusTransitions encodes the flow in PRD section 6.3.
// "diproses" is only reachable from "menunggu_pembayaran" — the hard rule
// that payment must be confirmed first — and cancellation is only offered
// at the two points the PRD's flow diagram actually shows it (before
// payment is confirmed). "selesai" and "dibatalkan" are terminal.
var AllowedOrderStatusTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusMenungguKonfirmasi: {OrderStatusMenungguPembayaran, OrderStatusDibatalkan},
	OrderStatusMenungguPembayaran: {OrderStatusDiproses, OrderStatusDibatalkan},
	OrderStatusDiproses:           {OrderStatusDikirim},
	OrderStatusDikirim:            {OrderStatusSelesai},
}

// IsTerminalOrderStatus reports whether an order in this status can still
// be modified (status changed, shipping_cost edited) — false once
// "selesai" or "dibatalkan", to protect data already counted in financial
// reports (PRD section 6.9).
func IsTerminalOrderStatus(status OrderStatus) bool {
	return status == OrderStatusSelesai || status == OrderStatusDibatalkan
}

// CanTransitionOrderStatus reports whether from -> to is an allowed order
// status transition.
func CanTransitionOrderStatus(from, to OrderStatus) bool {
	for _, allowed := range AllowedOrderStatusTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// NextOrderStatuses returns the statuses reachable from the given status,
// always as a non-nil slice (json.Marshal renders a nil slice as `null`,
// which would break frontend code expecting an array to call .length on).
func NextOrderStatuses(status OrderStatus) []OrderStatus {
	next := AllowedOrderStatusTransitions[status]
	if next == nil {
		return []OrderStatus{}
	}
	return next
}
