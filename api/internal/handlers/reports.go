package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
)

// parseReportPeriod reads required ?from=YYYY-MM-DD&to=YYYY-MM-DD (an
// inclusive day range) and an optional ?channel=online|pos filter shared
// by all three report endpoints. `to` is returned as an exclusive upper
// bound (the day after) so callers can just use created_at >= from AND
// created_at < to. Day boundaries are UTC — a reasonable MVP
// simplification given the PRD doesn't specify a store timezone.
func parseReportPeriod(r *http.Request) (from, to time.Time, channel *string, ok bool) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	fromDay, err1 := time.Parse("2006-01-02", fromStr)
	toDay, err2 := time.Parse("2006-01-02", toStr)
	if err1 != nil || err2 != nil || toDay.Before(fromDay) {
		return time.Time{}, time.Time{}, nil, false
	}

	from = fromDay
	to = toDay.AddDate(0, 0, 1)

	if c := r.URL.Query().Get("channel"); c == "online" || c == "pos" {
		channel = &c
	}

	return from, to, channel, true
}

func channelFilterLabel(channel *string) string {
	if channel == nil {
		return "all"
	}
	return *channel
}

type reportPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type orderCounts struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
}

type channelRevenue struct {
	Online int64 `json:"online"`
	POS    int64 `json:"pos"`
}

type reportRevenue struct {
	Gross         int64           `json:"gross"`
	DiscountTotal int64           `json:"discount_total"`
	ShippingTotal int64           `json:"shipping_total"`
	Net           int64           `json:"net"`
	ByChannel     *channelRevenue `json:"by_channel,omitempty"`
}

type reportSummaryResponse struct {
	Period        reportPeriod  `json:"period"`
	ChannelFilter string        `json:"channel_filter"`
	OrderCounts   orderCounts   `json:"order_counts"`
	Revenue       reportRevenue `json:"revenue"`
}

// ReportSummary handles GET /reports/summary — Owner only (PRD section
// 6.9). Only "selesai" orders count as revenue; discount_amount reduces
// net income, shipping_cost does not (it's passed straight through to the
// courier, not kept by the store).
func ReportSummary(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, channel, ok := parseReportPeriod(r)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "from/to wajib diisi format YYYY-MM-DD, dan to >= from")
			return
		}

		ctx := r.Context()
		var resp reportSummaryResponse
		resp.Period = reportPeriod{From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to")}
		resp.ChannelFilter = channelFilterLabel(channel)
		resp.OrderCounts.ByStatus = map[string]int64{}

		rows, err := pool.Query(ctx, `
			SELECT status::text, count(*)
			FROM orders
			WHERE created_at >= $1 AND created_at < $2
			  AND ($3::text IS NULL OR channel::text = $3)
			GROUP BY status
		`, from, to, channel)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load order counts")
			return
		}
		for rows.Next() {
			var status string
			var count int64
			if err := rows.Scan(&status, &count); err != nil {
				rows.Close()
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read order counts")
				return
			}
			resp.OrderCounts.ByStatus[status] = count
			resp.OrderCounts.Total += count
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read order counts")
			return
		}

		err = pool.QueryRow(ctx, `
			SELECT coalesce(sum(total), 0), coalesce(sum(discount_amount), 0)
			FROM orders
			WHERE status = 'selesai' AND created_at >= $1 AND created_at < $2
			  AND ($3::text IS NULL OR channel::text = $3)
		`, from, to, channel).Scan(&resp.Revenue.Gross, &resp.Revenue.DiscountTotal)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load revenue")
			return
		}
		resp.Revenue.Net = resp.Revenue.Gross - resp.Revenue.DiscountTotal

		// Shipping only ever applies to the online channel (PRD 6.9: "khusus
		// channel Online — transaksi POS tidak ada ongkir") — scoped to
		// 'online' regardless of the channel filter, since a 'pos' filter
		// would trivially yield 0 anyway.
		err = pool.QueryRow(ctx, `
			SELECT coalesce(sum(shipping_cost), 0)
			FROM orders
			WHERE status = 'selesai' AND channel = 'online' AND created_at >= $1 AND created_at < $2
		`, from, to).Scan(&resp.Revenue.ShippingTotal)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load shipping total")
			return
		}

		// PRD 6.9: "Revenue per channel ... ditampilkan terpisah di bawah
		// total gabungan" — only meaningful in the combined (unfiltered)
		// view; when already filtered to one channel it'd be redundant.
		if channel == nil {
			var byChannel channelRevenue
			channelRows, err := pool.Query(ctx, `
				SELECT channel::text, coalesce(sum(total), 0)
				FROM orders
				WHERE status = 'selesai' AND created_at >= $1 AND created_at < $2
				GROUP BY channel
			`, from, to)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load revenue by channel")
				return
			}
			for channelRows.Next() {
				var ch string
				var amount int64
				if err := channelRows.Scan(&ch, &amount); err != nil {
					channelRows.Close()
					httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read revenue by channel")
					return
				}
				if ch == "online" {
					byChannel.Online = amount
				} else {
					byChannel.POS = amount
				}
			}
			channelRows.Close()
			if err := channelRows.Err(); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read revenue by channel")
				return
			}
			resp.Revenue.ByChannel = &byChannel
		}

		httpx.WriteJSON(w, http.StatusOK, resp)
	}
}

type topProductRow struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	VariantID    string `json:"variant_id"`
	VariantName  string `json:"variant_name"`
	QuantitySold int64  `json:"quantity_sold"`
	Revenue      int64  `json:"revenue"`
}

// ReportTopProducts handles GET /reports/top-products — Owner only.
// Combines online + POS (PRD 6.9: "gabungan online + POS"), still
// respecting the shared ?channel= filter when the Owner narrows the view.
func ReportTopProducts(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, channel, ok := parseReportPeriod(r)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "from/to wajib diisi format YYYY-MM-DD, dan to >= from")
			return
		}
		limit := parsePositiveInt(r.URL.Query().Get("limit"), 10)
		if limit > 50 {
			limit = 50
		}

		rows, err := pool.Query(r.Context(), `
			SELECT p.id, p.name, pv.id, pv.variant_name,
			       sum(oi.quantity) AS qty_sold,
			       sum(oi.quantity * oi.price_at_purchase) AS revenue
			FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			JOIN product_variants pv ON pv.id = oi.product_variant_id
			JOIN products p ON p.id = pv.product_id
			WHERE o.status = 'selesai' AND o.created_at >= $1 AND o.created_at < $2
			  AND ($3::text IS NULL OR o.channel::text = $3)
			GROUP BY p.id, p.name, pv.id, pv.variant_name
			ORDER BY qty_sold DESC, revenue DESC
			LIMIT $4
		`, from, to, channel, limit)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load top products")
			return
		}
		defer rows.Close()

		products := []topProductRow{}
		for rows.Next() {
			var p topProductRow
			if err := rows.Scan(&p.ProductID, &p.ProductName, &p.VariantID, &p.VariantName, &p.QuantitySold, &p.Revenue); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read top products")
				return
			}
			products = append(products, p)
		}
		if err := rows.Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read top products")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": products})
	}
}

type salesTrendDay struct {
	Date          string `json:"date"`
	TotalOrders   int64  `json:"total_orders"`
	TotalRevenue  int64  `json:"total_revenue"`
	OnlineOrders  *int64 `json:"online_orders,omitempty"`
	OnlineRevenue *int64 `json:"online_revenue,omitempty"`
	POSOrders     *int64 `json:"pos_orders,omitempty"`
	POSRevenue    *int64 `json:"pos_revenue,omitempty"`
}

// ReportSalesTrend handles GET /reports/sales-trend — Owner only. Always
// returns one row per calendar day in the range (zero-filled), so the
// frontend chart has a continuous x-axis. When ?channel= is not set, also
// includes the online/pos split per day so the frontend can toggle the
// PRD's "dua garis: Online vs POS" view without a second request.
func ReportSalesTrend(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, channel, ok := parseReportPeriod(r)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "from/to wajib diisi format YYYY-MM-DD, dan to >= from")
			return
		}

		rows, err := pool.Query(r.Context(), `
			SELECT date_trunc('day', created_at)::date AS day, channel::text,
			       count(*) AS order_count, coalesce(sum(total), 0) AS revenue
			FROM orders
			WHERE status = 'selesai' AND created_at >= $1 AND created_at < $2
			  AND ($3::text IS NULL OR channel::text = $3)
			GROUP BY day, channel
			ORDER BY day
		`, from, to, channel)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load sales trend")
			return
		}
		defer rows.Close()

		type dayChannelAgg struct {
			orders  int64
			revenue int64
		}
		byDay := map[string]map[string]dayChannelAgg{}
		for rows.Next() {
			var day time.Time
			var ch string
			var agg dayChannelAgg
			if err := rows.Scan(&day, &ch, &agg.orders, &agg.revenue); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read sales trend")
				return
			}
			key := day.Format("2006-01-02")
			if byDay[key] == nil {
				byDay[key] = map[string]dayChannelAgg{}
			}
			byDay[key][ch] = agg
		}
		if err := rows.Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read sales trend")
			return
		}

		trend := []salesTrendDay{}
		for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
			key := d.Format("2006-01-02")
			channels := byDay[key]
			online := channels["online"]
			pos := channels["pos"]

			day := salesTrendDay{
				Date:         key,
				TotalOrders:  online.orders + pos.orders,
				TotalRevenue: online.revenue + pos.revenue,
			}
			if channel == nil {
				onlineOrders, onlineRevenue := online.orders, online.revenue
				posOrders, posRevenue := pos.orders, pos.revenue
				day.OnlineOrders = &onlineOrders
				day.OnlineRevenue = &onlineRevenue
				day.POSOrders = &posOrders
				day.POSRevenue = &posRevenue
			}
			trend = append(trend, day)
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": trend})
	}
}
