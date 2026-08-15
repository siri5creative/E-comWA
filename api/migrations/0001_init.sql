-- 0001_init.sql
-- Initial schema for the shared e-commerce + POS database.
-- Source of truth: prd-ecommerce-wa.md section 7 (schema) and section 7A (POS sharing rules).
-- Run once in the Supabase SQL Editor on a fresh project.

-- ============================================================================
-- Extensions
-- ============================================================================

create extension if not exists pgcrypto; -- gen_random_uuid()

-- ============================================================================
-- Enum types
-- ============================================================================

create type admin_role as enum ('owner', 'staff');
create type coupon_discount_type as enum ('total_belanja', 'item_tertentu', 'event', 'bundle');
create type order_channel as enum ('online', 'pos');
create type order_status as enum (
  'menunggu_konfirmasi',
  'menunggu_pembayaran',
  'diproses',
  'dikirim',
  'selesai',
  'dibatalkan'
);
create type payment_provider as enum ('midtrans', 'xendit');

-- ============================================================================
-- Tables
-- ============================================================================

-- customers: lightweight, no auth/password. whatsapp_number is stored in
-- international format (62xxx), no leading "+" or "0" — see PRD section 11.
create table customers (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  whatsapp_number text not null unique,
  created_at timestamptz not null default now()
);

-- admins: e-commerce dashboard accounts only (Owner/Staff), backed by
-- Supabase Auth. Entirely separate from the POS app's own cashier login.
create table admins (
  id uuid primary key default gen_random_uuid(),
  auth_user_id uuid not null unique references auth.users (id) on delete cascade,
  name text not null,
  role admin_role not null,
  created_at timestamptz not null default now()
);

create table categories (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null unique
);

create table products (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null unique,
  description text,
  category_id uuid references categories (id) on delete set null,
  cover_image_url text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- product_variants: size/color variants, stock tracked per variant.
-- stock_quantity must never go negative — enforced both here and by the
-- atomic decrement query the backend uses (PRD section 7A).
create table product_variants (
  id uuid primary key default gen_random_uuid(),
  product_id uuid not null references products (id) on delete cascade,
  variant_name text not null,
  sku text unique,
  price numeric(12, 2) not null check (price >= 0),
  stock_quantity integer not null default 0 check (stock_quantity >= 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table coupons (
  id uuid primary key default gen_random_uuid(),
  code text not null unique,
  discount_type coupon_discount_type not null,
  discount_value numeric(12, 2) not null check (discount_value >= 0),
  valid_from timestamptz not null,
  valid_until timestamptz not null,
  max_total_usage integer,
  max_usage_per_customer integer,
  current_usage_count integer not null default 0,
  is_active boolean not null default true,
  created_by uuid references admins (id) on delete set null,
  created_at timestamptz not null default now(),
  check (valid_until >= valid_from)
);

-- coupon_products: used only for discount_type 'item_tertentu' or 'bundle'.
create table coupon_products (
  id uuid primary key default gen_random_uuid(),
  coupon_id uuid not null references coupons (id) on delete cascade,
  product_id uuid not null references products (id) on delete cascade
);

-- orders
-- channel defaults to 'online' because checkout is the default entry point;
-- the POS handler must explicitly set channel = 'pos'.
-- status 'diproses' must only be reachable after an admin manually confirms
-- payment in full — this invariant is enforced in application code, not here
-- (PRD section 6.3).
create table orders (
  id uuid primary key default gen_random_uuid(),
  invoice_number text not null unique,
  customer_id uuid references customers (id) on delete set null,
  channel order_channel not null default 'online',
  status order_status not null default 'menunggu_konfirmasi',
  coupon_id uuid references coupons (id) on delete set null,
  subtotal numeric(12, 2) not null default 0 check (subtotal >= 0),
  discount_amount numeric(12, 2) not null default 0 check (discount_amount >= 0),
  shipping_cost numeric(12, 2) not null default 0 check (shipping_cost >= 0),
  total numeric(12, 2) not null default 0 check (total >= 0),
  shipping_note text,
  payment_proof_note text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table order_items (
  id uuid primary key default gen_random_uuid(),
  order_id uuid not null references orders (id) on delete cascade,
  product_variant_id uuid not null references product_variants (id) on delete restrict,
  quantity integer not null check (quantity > 0),
  price_at_purchase numeric(12, 2) not null check (price_at_purchase >= 0)
);

-- coupon_usages: kept as history for reporting/audit. Rows older than 3
-- months are purged by a scheduled backend job (PRD section 11) — this is
-- safe because coupons.current_usage_count is the permanent running total
-- and does not depend on these rows.
create table coupon_usages (
  id uuid primary key default gen_random_uuid(),
  coupon_id uuid not null references coupons (id) on delete cascade,
  customer_id uuid not null references customers (id) on delete cascade,
  order_id uuid not null references orders (id) on delete cascade,
  used_at timestamptz not null default now()
);

-- admin_devices: FCM device tokens for push notifications.
create table admin_devices (
  id uuid primary key default gen_random_uuid(),
  admin_id uuid not null references admins (id) on delete cascade,
  fcm_device_token text not null unique,
  created_at timestamptz not null default now()
);

-- payment_gateway_settings: prepared for future use, not active yet
-- (PRD section 6.8) — encrypted_credentials must only ever be
-- read/written by the backend using PAYMENT_GATEWAY_ENCRYPTION_KEY.
create table payment_gateway_settings (
  id uuid primary key default gen_random_uuid(),
  provider payment_provider not null,
  is_sandbox boolean not null default true,
  encrypted_credentials text,
  is_active boolean not null default false,
  updated_by uuid references admins (id) on delete set null,
  updated_at timestamptz not null default now()
);

-- ============================================================================
-- Indexes
-- ============================================================================

create index idx_products_category_id on products (category_id);
create index idx_product_variants_product_id on product_variants (product_id);

create index idx_coupons_is_active on coupons (is_active);
create index idx_coupon_products_coupon_id on coupon_products (coupon_id);
create index idx_coupon_products_product_id on coupon_products (product_id);

-- Speeds up the per-customer usage-limit check at checkout (PRD section 6.4).
create index idx_coupon_usages_coupon_customer on coupon_usages (coupon_id, customer_id);
create index idx_coupon_usages_order_id on coupon_usages (order_id);
-- Speeds up the scheduled purge job (used_at < now() - interval '3 months').
create index idx_coupon_usages_used_at on coupon_usages (used_at);

create index idx_orders_customer_id on orders (customer_id);
create index idx_orders_coupon_id on orders (coupon_id);
create index idx_orders_status on orders (status);
create index idx_orders_channel on orders (channel);
create index idx_orders_created_at on orders (created_at);

create index idx_order_items_order_id on order_items (order_id);
create index idx_order_items_product_variant_id on order_items (product_variant_id);

create index idx_admin_devices_admin_id on admin_devices (admin_id);

-- ============================================================================
-- updated_at triggers
-- ============================================================================

create or replace function set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger trg_products_updated_at
  before update on products
  for each row execute function set_updated_at();

create trigger trg_product_variants_updated_at
  before update on product_variants
  for each row execute function set_updated_at();

create trigger trg_orders_updated_at
  before update on orders
  for each row execute function set_updated_at();

create trigger trg_payment_gateway_settings_updated_at
  before update on payment_gateway_settings
  for each row execute function set_updated_at();

-- ============================================================================
-- Helper functions for RLS
--
-- SECURITY DEFINER so the check itself can read `admins` regardless of the
-- caller's own RLS grants on that table (admins is owner-only to SELECT,
-- see policies below) — otherwise a staff admin's own "am I an admin?"
-- check would be blocked by the very table it's checking.
-- ============================================================================

create or replace function is_admin()
returns boolean
language sql
security definer
set search_path = public
stable
as $$
  select exists (
    select 1 from admins a where a.auth_user_id = auth.uid()
  );
$$;

create or replace function is_owner()
returns boolean
language sql
security definer
set search_path = public
stable
as $$
  select exists (
    select 1 from admins a where a.auth_user_id = auth.uid() and a.role = 'owner'
  );
$$;

-- ============================================================================
-- Row Level Security
--
-- Note: the Go backend connects using the Supabase service role key for
-- writes that must bypass RLS (see IMPLEMENTATION.md section 2), so these
-- policies are the defense-in-depth boundary for any direct/anon access,
-- per PRD section 7.
-- ============================================================================

alter table customers enable row level security;
alter table admins enable row level security;
alter table categories enable row level security;
alter table products enable row level security;
alter table product_variants enable row level security;
alter table coupons enable row level security;
alter table coupon_products enable row level security;
alter table coupon_usages enable row level security;
alter table orders enable row level security;
alter table order_items enable row level security;
alter table admin_devices enable row level security;
alter table payment_gateway_settings enable row level security;

-- Public: SELECT on products, product_variants, categories
create policy "public read categories" on categories for select using (true);
create policy "public read products" on products for select using (true);
create policy "public read product_variants" on product_variants for select using (true);

-- Public: INSERT on customers, orders, order_items (intended to be called
-- through the backend, not directly from a client — PRD section 7)
create policy "public insert customers" on customers for insert with check (true);
create policy "public insert orders" on orders for insert with check (true);
create policy "public insert order_items" on order_items for insert with check (true);

-- Admin (authenticated, any role): full access to orders, order_items,
-- product_variants (needed to update stock)
create policy "admin full access orders" on orders for all using (is_admin()) with check (is_admin());
create policy "admin full access order_items" on order_items for all using (is_admin()) with check (is_admin());
create policy "admin full access product_variants" on product_variants for all using (is_admin()) with check (is_admin());

-- Admin (any role) also manage products & categories (PRD section 8: POST/PUT/DELETE
-- /products is "Admin", not Owner-only).
create policy "admin full access products" on products for all using (is_admin()) with check (is_admin());
create policy "admin full access categories" on categories for all using (is_admin()) with check (is_admin());

-- Admin (any role): read/manage their own device tokens for push notifications
create policy "admin manage own devices" on admin_devices for all using (is_admin()) with check (is_admin());

-- Admin (role owner only): coupons, admins, payment_gateway_settings
create policy "owner full access coupons" on coupons for all using (is_owner()) with check (is_owner());
create policy "owner full access coupon_products" on coupon_products for all using (is_owner()) with check (is_owner());
create policy "owner full access coupon_usages" on coupon_usages for all using (is_owner()) with check (is_owner());
create policy "owner full access admins" on admins for all using (is_owner()) with check (is_owner());
create policy "owner full access payment_gateway_settings" on payment_gateway_settings for all using (is_owner()) with check (is_owner());

-- Public: validate a coupon code at checkout (read-only, active coupons only).
-- Needed for POST /coupons/validate (PRD section 8) without granting broad
-- coupon read access to anonymous clients.
create policy "public read active coupons" on coupons for select using (is_active = true);
