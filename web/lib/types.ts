// Mirrors the JSON shapes produced by the Go backend
// (api/internal/models, api/internal/handlers).

export type Category = {
  id: string;
  name: string;
  slug: string;
};

export type ProductVariant = {
  id: string;
  variant_name: string;
  sku: string | null;
  price: number;
  stock_quantity: number;
};

export type Product = {
  id: string;
  name: string;
  slug: string;
  description: string | null;
  cover_image_url: string | null;
  category: Category | null;
  variants: ProductVariant[];
};

export type ProductListResponse = {
  data: Product[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
};

export type CategoryListResponse = {
  data: Category[];
};

export type CheckoutItemRequest = {
  product_variant_id: string;
  quantity: number;
};

export type CheckoutRequest = {
  name: string;
  whatsapp_number: string;
  items: CheckoutItemRequest[];
};

export type OrderStatus =
  | "menunggu_konfirmasi"
  | "menunggu_pembayaran"
  | "diproses"
  | "dikirim"
  | "selesai"
  | "dibatalkan";

export type CheckoutOrder = {
  order_id: string;
  invoice_number: string;
  channel: "online" | "pos";
  status: OrderStatus;
  coupon_code?: string;
  subtotal: number;
  discount_amount: number;
  shipping_cost: number;
  total: number;
  created_at: string;
};

export type ApiError = {
  error: string;
  message: string;
};

export type InsufficientStockDetail = {
  product_variant_id: string;
  requested: number;
  available: number;
};

export type InsufficientStockError = ApiError & {
  details: InsufficientStockDetail[];
};

export type CouponInvalidError = ApiError & {
  reason: string;
};

// --- Admin (order management) ---

export type AdminRole = "owner" | "staff";

export type AdminMe = {
  id: string;
  name: string;
  role: AdminRole;
  created_at: string;
};

export type OrderListItem = {
  id: string;
  invoice_number: string;
  channel: "online" | "pos";
  status: OrderStatus;
  customer_name: string | null;
  customer_whatsapp: string | null;
  subtotal: number;
  discount_amount: number;
  shipping_cost: number;
  total: number;
  created_at: string;
};

export type OrderListResponse = {
  data: OrderListItem[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
};

export type OrderDetailItem = {
  product_variant_id: string;
  product_name: string;
  variant_name: string;
  quantity: number;
  price_at_purchase: number;
};

export type OrderDetail = {
  id: string;
  invoice_number: string;
  channel: "online" | "pos";
  status: OrderStatus;
  next_statuses: OrderStatus[];
  customer: { name: string; whatsapp_number: string } | null;
  subtotal: number;
  discount_amount: number;
  shipping_cost: number;
  total: number;
  shipping_note: string | null;
  payment_proof_note: string | null;
  items: OrderDetailItem[];
  created_at: string;
  updated_at: string;
};

export type UpdateOrderResponse = {
  id: string;
  status: OrderStatus;
  shipping_cost: number;
  total: number;
  next_statuses: OrderStatus[];
};

export type WhatsAppMessageResponse = {
  whatsapp_number: string;
  message: string;
  wa_link: string;
};

// --- Coupons ---

export type CouponDiscountType =
  | "total_belanja"
  | "item_tertentu"
  | "event"
  | "bundle";

export type CouponDiscountValueType = "fixed" | "percentage";

export type Coupon = {
  id: string;
  code: string;
  discount_type: CouponDiscountType;
  discount_value_type: CouponDiscountValueType;
  discount_value: number;
  min_spend: number;
  valid_from: string;
  valid_until: string;
  max_total_usage: number | null;
  max_usage_per_customer: number | null;
  current_usage_count: number;
  is_active: boolean;
  product_ids?: string[];
  created_at: string;
};

export type ValidateCouponResponse =
  | {
      valid: true;
      code: string;
      discount_type: CouponDiscountType;
      discount_amount: number;
    }
  | { valid: false; reason: string; message: string };

// --- Reports ---

export type ChannelFilter = "all" | "online" | "pos";

export type ReportSummary = {
  period: { from: string; to: string };
  channel_filter: ChannelFilter;
  order_counts: {
    total: number;
    by_status: Partial<Record<OrderStatus, number>>;
  };
  revenue: {
    gross: number;
    discount_total: number;
    shipping_total: number;
    net: number;
    by_channel?: { online: number; pos: number };
  };
};

export type TopProduct = {
  product_id: string;
  product_name: string;
  variant_id: string;
  variant_name: string;
  quantity_sold: number;
  revenue: number;
};

export type SalesTrendDay = {
  date: string;
  total_orders: number;
  total_revenue: number;
  online_orders?: number;
  online_revenue?: number;
  pos_orders?: number;
  pos_revenue?: number;
};

// --- Payment gateway settings (prepared, not active — PRD 6.8) ---

export type PaymentProvider = "midtrans" | "xendit";

export type PaymentSettings = {
  configured: boolean;
  provider?: PaymentProvider;
  is_sandbox?: boolean;
  is_active?: boolean;
  has_credentials: boolean;
  updated_at?: string;
};

export type TestPaymentConnectionResponse = {
  success: boolean;
  message: string;
};
