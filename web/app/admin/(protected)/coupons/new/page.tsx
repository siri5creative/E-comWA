import { goFetch } from "@/lib/api";
import type { ProductListResponse } from "@/lib/types";
import { OwnerOnly } from "@/components/admin/OwnerOnly";
import { CouponForm } from "@/components/admin/CouponForm";

async function getProducts() {
  const res = await goFetch("/products?page_size=100");
  if (!res.ok) {
    throw new Error("Gagal memuat produk");
  }
  const body = (await res.json()) as ProductListResponse;
  return body.data;
}

export default async function NewCouponPage() {
  return (
    <OwnerOnly>
      <NewCouponContent />
    </OwnerOnly>
  );
}

async function NewCouponContent() {
  const products = await getProducts();

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <h1 className="mb-8 text-2xl font-black uppercase tracking-tight">
        Tambah Kupon
      </h1>
      <CouponForm mode="create" products={products} />
    </div>
  );
}
