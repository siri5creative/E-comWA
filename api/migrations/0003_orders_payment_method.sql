-- 0003_orders_payment_method.sql
-- api-pos-integration.md section 5.3 has POST /pos/orders accept a
-- payment_method field ("untuk keperluan laporan saja") and section 5.4
-- returns it in the transaction detail — but the orders table in PRD
-- section 7 has no column for it. Adding one so it's actually persisted
-- rather than silently discarded. Nullable and free-text (the docs say
-- "bebas string, contoh cash/qris/debit" — not a fixed enum) — only ever
-- set by POS orders; online checkout leaves it null.
-- Run after 0001_init.sql and 0002_coupon_discount_fields.sql.

alter table orders
  add column payment_method text;
