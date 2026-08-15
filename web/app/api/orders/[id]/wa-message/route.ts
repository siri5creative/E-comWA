import { NextRequest, NextResponse } from "next/server";
import { goFetchAsAdmin } from "@/lib/api";

export async function GET(
  req: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const res = await goFetchAsAdmin(`/orders/${id}/wa-message`);
  const body = await res.text();
  return new NextResponse(body, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
