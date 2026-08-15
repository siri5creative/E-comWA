import { NextRequest, NextResponse } from "next/server";
import { goFetchAsAdmin } from "@/lib/api";

export async function PATCH(
  req: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const body = await req.text();
  const res = await goFetchAsAdmin(`/orders/${id}/status`, {
    method: "PATCH",
    body,
  });
  const responseBody = await res.text();
  return new NextResponse(responseBody, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
