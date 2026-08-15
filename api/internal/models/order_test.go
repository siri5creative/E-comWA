package models

import "testing"

func TestCanTransitionOrderStatus(t *testing.T) {
	cases := []struct {
		name string
		from OrderStatus
		to   OrderStatus
		want bool
	}{
		{"cannot skip payment confirmation straight to diproses", OrderStatusMenungguKonfirmasi, OrderStatusDiproses, false},
		{"menunggu_konfirmasi to menunggu_pembayaran is allowed", OrderStatusMenungguKonfirmasi, OrderStatusMenungguPembayaran, true},
		{"menunggu_konfirmasi to dibatalkan is allowed", OrderStatusMenungguKonfirmasi, OrderStatusDibatalkan, true},
		{"diproses only reachable after payment confirmed", OrderStatusMenungguPembayaran, OrderStatusDiproses, true},
		{"menunggu_pembayaran to dibatalkan is allowed", OrderStatusMenungguPembayaran, OrderStatusDibatalkan, true},
		{"diproses to dikirim is allowed", OrderStatusDiproses, OrderStatusDikirim, true},
		{"diproses cannot be cancelled (not in PRD's flow diagram)", OrderStatusDiproses, OrderStatusDibatalkan, false},
		{"dikirim to selesai is allowed", OrderStatusDikirim, OrderStatusSelesai, true},
		{"selesai is terminal, no transitions out", OrderStatusSelesai, OrderStatusDibatalkan, false},
		{"dibatalkan is terminal, no transitions out", OrderStatusDibatalkan, OrderStatusMenungguPembayaran, false},
		{"cannot jump backwards", OrderStatusDikirim, OrderStatusMenungguPembayaran, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanTransitionOrderStatus(tc.from, tc.to)
			if got != tc.want {
				t.Errorf("CanTransitionOrderStatus(%s, %s) = %v; want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestIsTerminalOrderStatus(t *testing.T) {
	terminal := []OrderStatus{OrderStatusSelesai, OrderStatusDibatalkan}
	for _, s := range terminal {
		if !IsTerminalOrderStatus(s) {
			t.Errorf("IsTerminalOrderStatus(%s) = false; want true", s)
		}
	}

	nonTerminal := []OrderStatus{
		OrderStatusMenungguKonfirmasi,
		OrderStatusMenungguPembayaran,
		OrderStatusDiproses,
		OrderStatusDikirim,
	}
	for _, s := range nonTerminal {
		if IsTerminalOrderStatus(s) {
			t.Errorf("IsTerminalOrderStatus(%s) = true; want false", s)
		}
	}
}

func TestNextOrderStatusesNeverReturnsNil(t *testing.T) {
	// json.Marshal renders a nil slice as `null`, which breaks frontend
	// code calling .length on it — this must always be a non-nil slice,
	// even for terminal statuses that have no next steps.
	for _, s := range []OrderStatus{OrderStatusSelesai, OrderStatusDibatalkan} {
		next := NextOrderStatuses(s)
		if next == nil {
			t.Errorf("NextOrderStatuses(%s) = nil; want non-nil empty slice", s)
		}
		if len(next) != 0 {
			t.Errorf("NextOrderStatuses(%s) = %v; want empty", s, next)
		}
	}

	next := NextOrderStatuses(OrderStatusMenungguKonfirmasi)
	if len(next) != 2 {
		t.Errorf("NextOrderStatuses(%s) = %v; want 2 entries", OrderStatusMenungguKonfirmasi, next)
	}
}

func TestValidOrderStatusesMatchesTransitionKeys(t *testing.T) {
	// Every status referenced anywhere in AllowedOrderStatusTransitions
	// (as either a source or a destination) must be a recognized status.
	for from, tos := range AllowedOrderStatusTransitions {
		if !ValidOrderStatuses[from] {
			t.Errorf("AllowedOrderStatusTransitions has unknown source status %s", from)
		}
		for _, to := range tos {
			if !ValidOrderStatuses[to] {
				t.Errorf("AllowedOrderStatusTransitions[%s] has unknown destination status %s", from, to)
			}
		}
	}
}
