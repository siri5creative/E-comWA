# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Status

This repository currently contains **only planning documents** in `files/` — no application code exists yet (no `web/`, no `api/`, no `package.json`, no `go.mod`). Before writing any code, read the specs in this order:

1. `files/IMPLEMENTATION.md` — entry point; how to build it in this repo (monorepo layout, env vars, deployment plan, priority order)
2. `files/prd-ecommerce-wa.md` — business logic, full DB schema, API spec, non-functional requirements
3. `files/api-pos-integration.md` — API contract specifically for the pre-existing POS app
4. `files/design-brief-ecommerce-nike-style.md` — visual direction for `web/`

Once `web/` and `api/` exist, this file should be updated with real build/lint/test commands (`npm run dev`, `go run main.go`, etc., per `files/IMPLEMENTATION.md` section 2/5).

## What This Project Is

An e-commerce site for a small UMKM (micro/small business) seller. Payment confirmation and order-status updates happen manually via WhatsApp chat (not the WhatsApp Business API) — orders are never auto-processed or auto-cancelled by the system. The distinguishing architectural fact: this database is **shared as the single source of truth with an already-existing POS app** used at the physical store, not a database built for e-commerce alone.

## Intended Monorepo Structure

Per `files/IMPLEMENTATION.md` section 1:

```
web/    Next.js (App Router, TypeScript, Tailwind) — public storefront + admin dashboard
api/    Go REST API — internal/handlers, internal/middleware, internal/models, internal/db, migrations/
```

Deployed together as one Vercel project using Vercel Services (`web/` on the Next.js preset, `api/` on the Go preset).

## Critical Architectural Constraints

These are non-obvious rules that span multiple files/layers — violating them breaks the shared-database contract with the POS app or the payment-verification flow.

- **All stock writes go through the Go backend, from both web and POS.** Neither app may write to Supabase directly. Stock decrements must be a single atomic SQL statement (`UPDATE product_variants SET stock_quantity = stock_quantity - :qty WHERE id = :id AND stock_quantity >= :qty`), never a separate check-then-update, to prevent race conditions between a simultaneous online order and an in-store POS sale on the same variant.
- **POS auth is separate from admin auth.** Admin endpoints use Supabase Auth (email+password) with `owner`/`staff` role checks enforced in the backend. `/pos/*` endpoints instead validate a static `POS_API_KEY` sent as `Authorization: Bearer <key>` — POS keeps its own separate cashier login system that this repo does not implement or touch. See `files/api-pos-integration.md` for the full POS-facing contract.
- **Order status `diproses` (processing) can only be set after an admin manually marks payment as fully confirmed.** This is a hard rule — never allow a code path that skips this. There is no automatic order cancellation for unpaid orders past a deadline; follow-up is always manual by an admin.
- **Shipping cost (`shipping_cost`) is never calculated automatically.** It's agreed manually over WhatsApp, then an admin must manually enter it into the order in the dashboard for it to be persisted and included in the order total / financial reports. POS transactions never have a `shipping_cost`.
- **POS transactions bypass the online order lifecycle entirely**: they're created directly with status `selesai` (completed), don't require a `customer_id`, and don't support coupons (coupon support for POS is explicitly undecided — see `files/api-pos-integration.md` section 7).
- **Role checks (`owner` vs `staff`) must be enforced in the Go backend on every sensitive endpoint**, not just hidden in the frontend UI. Coupon management, admin account management, payment-gateway settings, and financial reports are `owner`-only.
- **WhatsApp numbers are normalized to `62xxx` format** (international, no `+`, no leading `0`) before storage, per the `wa.me` link spec — conversion happens at the checkout form.
- **`coupon_usages` rows are purged every 3 months** via a scheduled backend job, while `coupons.current_usage_count` persists permanently as the source of truth for usage limits. This means "per-customer" coupon limits can become inaccurate for coupons valid longer than 3 months — see the caveat in `files/prd-ecommerce-wa.md` section 11.

## Env Vars

Full list and required secrecy rules are in `files/IMPLEMENTATION.md` section 2. Notably: `SUPABASE_SERVICE_ROLE_KEY`, `PAYMENT_GATEWAY_ENCRYPTION_KEY`, and `POS_API_KEY` must never reach frontend/browser code.

## Build Order

`files/IMPLEMENTATION.md` section 4 specifies a strict build order — don't jump ahead to a lower-priority feature before the foundation is done: DB schema/RLS → backend core (products, checkout, admin auth) → public frontend → admin order management → coupons → notifications (FCM push + `wa.me` links) → financial reports → payment-gateway settings UI (built but inactive) → POS integration endpoints (built last, since it depends on stock logic being stable).

## Open Decisions — Ask, Don't Assume

`files/prd-ecommerce-wa.md` section 11 flags several points as needing a decision during implementation (e.g., final POS auth mechanism, long-lived-coupon vs. 3-month `coupon_usages` retention conflict). When you hit one of these, ask the user rather than deciding unilaterally — this is an explicit instruction from `files/IMPLEMENTATION.md` section 5.
