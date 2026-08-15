import { notFound } from "next/navigation";
import { goFetch, goFetchAsAdmin } from "@/lib/api";
import type { Coupon, ProductListResponse } from "@/lib/types";
import { OwnerOnly } from "@/components/admin/OwnerOnly";
import { CouponForm } from "@/components/admin/CouponForm";

async function getCoupon(id: string): Promise<Coupon | null> {
  const res = await goFetchAsAdmin(`/coupons/${id}`);
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error("Gagal memuat kupon");
  }
  return (await res.json()) as Coupon;
}

async function getProducts() {
  const res = await goFetch("/products?page_size=100");
  if (!res.ok) {
    throw new Error("Gagal memuat produk");
  }
  const body = (await res.json()) as ProductListResponse;
  return body.data;
}

export default async function EditCouponPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  return (
    <OwnerOnly>
      <EditCouponContent id={id} />
    </OwnerOnly>
  );
}

async function EditCouponContent({ id }: { id: string }) {
  const [coupon, products] = await Promise.all([getCoupon(id), getProducts()]);

  if (!coupon) {
    notFound();
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <h1 className="mb-8 text-2xl font-black uppercase tracking-tight">
        Edit Kupon
      </h1>
      <CouponForm mode="edit" couponId={coupon.id} initial={coupon} products={products} />
    </div>
  );
}
