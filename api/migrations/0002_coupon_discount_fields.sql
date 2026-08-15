-- 0002_coupon_discount_fields.sql
-- Extends coupons with fields needed for real discount calculation, decided
-- with the user beyond what prd-ecommerce-wa.md section 7 specified:
--   - discount_value_type: discount_value can be a flat Rupiah amount or a
--     percentage, chosen per coupon (independent of discount_type, which is
--     the business category: total_belanja/item_tertentu/event/bundle).
--   - min_spend: optional minimum cart subtotal required to use the coupon
--     (the PRD's "belanja di atas Rp100rb" example needs this, and the
--     original schema had no column for it).
-- Run after 0001_init.sql.

create type coupon_discount_value_type as enum ('fixed', 'percentage');

alter table coupons
  add column discount_value_type coupon_discount_value_type not null default 'fixed',
  add column min_spend numeric(12, 2) not null default 0 check (min_spend >= 0);

-- A percentage only makes sense as a ratio 0-100; a flat amount has no such
-- ceiling (validated as >= 0 already by the original discount_value check).
alter table coupons
  add constraint coupons_percentage_range_chk
  check (discount_value_type <> 'percentage' OR (discount_value > 0 AND discount_value <= 100));
