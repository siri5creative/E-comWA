package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/joho/godotenv"

	"github.com/siri5creative/E-comWA/api/internal/config"
	"github.com/siri5creative/E-comWA/api/internal/crypto"
	"github.com/siri5creative/E-comWA/api/internal/db"
	"github.com/siri5creative/E-comWA/api/internal/handlers"
	"github.com/siri5creative/E-comWA/api/internal/middleware"
	"github.com/siri5creative/E-comWA/api/internal/notify"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Local-dev convenience only; in deployment env vars are set directly
	// (Vercel dashboard), so a missing .env file here is not an error.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// nil (notifications disabled) when FIREBASE_SERVICE_ACCOUNT_KEY isn't
	// set yet — this is an enhancement, not required to run the app.
	fcm, err := notify.NewFCMClient(ctx, cfg.FirebaseServiceAccountKey)
	if err != nil {
		return err
	}
	if fcm == nil {
		slog.Warn("FIREBASE_SERVICE_ACCOUNT_KEY not set — admin push notifications disabled")
	}

	// nil (saving disabled, 503) when PAYMENT_GATEWAY_ENCRYPTION_KEY isn't
	// set — this feature is explicitly "disiapkan, belum aktif" (PRD 6.8).
	paymentBox, err := crypto.NewBox(cfg.PaymentGatewayEncryptionKey)
	if err != nil {
		return err
	}
	if paymentBox == nil {
		slog.Warn("PAYMENT_GATEWAY_ENCRYPTION_KEY not set — saving payment settings disabled")
	}

	if cfg.POSAPIKey == "" {
		slog.Warn("POS_API_KEY not set — /pos/* endpoints will reject all requests")
	}

	// Projects using Supabase's newer asymmetric "JWT Signing Keys" sign
	// Auth tokens with ES256, verified against the project's published
	// JWKS rather than a shared secret — see internal/middleware/auth_admin.go.
	var jwks keyfunc.Keyfunc
	if cfg.SupabaseURL != "" {
		jwksURL := strings.TrimRight(cfg.SupabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
		jwks, err = keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
		if err != nil {
			return fmt.Errorf("fetch Supabase JWKS from %s: %w", jwksURL, err)
		}
	}

	adminAuth := &middleware.AdminAuth{Pool: pool, JWTSecret: cfg.SupabaseJWTSecret, JWKS: jwks}
	posAuth := middleware.RequirePOSAPIKey(cfg.POSAPIKey)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /api/v1/categories", handlers.ListCategories(pool))
	mux.HandleFunc("GET /api/v1/products", handlers.ListProducts(pool))
	mux.HandleFunc("GET /api/v1/products/{slug}", handlers.GetProductBySlug(pool))
	mux.HandleFunc("GET /api/v1/products/variants/{variant_id}/stock", handlers.GetVariantStock(pool))
	mux.HandleFunc("POST /api/v1/checkout", handlers.Checkout(pool, fcm))

	// Product/variant/stock management — PRD section 6.7: any admin role
	// (not Owner-only), unlike coupons/reports/payment-settings below.
	mux.Handle("POST /api/v1/products", adminAuth.RequireAdmin(handlers.CreateProduct(pool)))
	mux.Handle("PUT /api/v1/products/{id}", adminAuth.RequireAdmin(handlers.UpdateProduct(pool)))
	mux.Handle("DELETE /api/v1/products/{id}", adminAuth.RequireAdmin(handlers.DeleteProduct(pool)))
	mux.Handle("POST /api/v1/products/{id}/variants", adminAuth.RequireAdmin(handlers.CreateVariant(pool)))
	mux.Handle("PUT /api/v1/products/variants/{variant_id}", adminAuth.RequireAdmin(handlers.UpdateVariant(pool)))
	mux.Handle("DELETE /api/v1/products/variants/{variant_id}", adminAuth.RequireAdmin(handlers.DeleteVariant(pool)))

	mux.HandleFunc("POST /api/v1/coupons/validate", handlers.ValidateCoupon(pool))

	mux.Handle("POST /api/v1/pos/orders", posAuth(handlers.CreatePOSOrder(pool)))
	mux.Handle("GET /api/v1/pos/orders/{order_id}", posAuth(handlers.GetPOSOrder(pool)))

	mux.Handle("GET /api/v1/admin/me", adminAuth.RequireAdmin(handlers.Me()))
	mux.Handle("POST /api/v1/admin-devices", adminAuth.RequireAdmin(handlers.RegisterAdminDevice(pool)))

	mux.Handle("GET /api/v1/orders", adminAuth.RequireAdmin(handlers.ListOrders(pool)))
	mux.Handle("GET /api/v1/orders/{id}", adminAuth.RequireAdmin(handlers.GetOrder(pool)))
	mux.Handle("PATCH /api/v1/orders/{id}/status", adminAuth.RequireAdmin(handlers.UpdateOrderStatus(pool)))
	mux.Handle("GET /api/v1/orders/{id}/wa-message", adminAuth.RequireAdmin(handlers.GenerateWAMessage(pool)))

	mux.Handle("GET /api/v1/coupons", adminAuth.RequireOwner(handlers.ListCoupons(pool)))
	mux.Handle("GET /api/v1/coupons/{id}", adminAuth.RequireOwner(handlers.GetCoupon(pool)))
	mux.Handle("POST /api/v1/coupons", adminAuth.RequireOwner(handlers.CreateCoupon(pool)))
	mux.Handle("PUT /api/v1/coupons/{id}", adminAuth.RequireOwner(handlers.UpdateCoupon(pool)))
	mux.Handle("DELETE /api/v1/coupons/{id}", adminAuth.RequireOwner(handlers.DeleteCoupon(pool)))

	mux.Handle("GET /api/v1/reports/summary", adminAuth.RequireOwner(handlers.ReportSummary(pool)))
	mux.Handle("GET /api/v1/reports/top-products", adminAuth.RequireOwner(handlers.ReportTopProducts(pool)))
	mux.Handle("GET /api/v1/reports/sales-trend", adminAuth.RequireOwner(handlers.ReportSalesTrend(pool)))

	mux.Handle("GET /api/v1/payment-settings", adminAuth.RequireOwner(handlers.GetPaymentSettings(pool)))
	mux.Handle("POST /api/v1/payment-settings", adminAuth.RequireOwner(handlers.SavePaymentSettings(pool, paymentBox)))
	mux.Handle("POST /api/v1/payment-settings/test", adminAuth.RequireOwner(handlers.TestPaymentConnection()))

	var h http.Handler = mux
	h = middleware.CORS(cfg.CORSAllowedOrigins)(h)
	h = middleware.Logging(h)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
