import { NextRequest, NextResponse } from "next/server";
import { goFetchAsAdmin } from "@/lib/api";

export async function PUT(
  req: NextRequest,
  { params }: { params: Promise<{ variant_id: string }> }
) {
  const { variant_id } = await params;
  const body = await req.text();
  const res = await goFetchAsAdmin(`/products/variants/${variant_id}`, {
    method: "PUT",
    body,
  });
  const responseBody = await res.text();
  return new NextResponse(responseBody, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}

export async function DELETE(
  req: NextRequest,
  { params }: { params: Promise<{ variant_id: string }> }
) {
  const { variant_id } = await params;
  const res = await goFetchAsAdmin(`/products/variants/${variant_id}`, {
    method: "DELETE",
  });
  if (res.status === 204) {
    return new NextResponse(null, { status: 204 });
  }
  const body = await res.text();
  return new NextResponse(body, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
