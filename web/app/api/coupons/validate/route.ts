import { NextRequest, NextResponse } from "next/server";
import { goFetch } from "@/lib/api";

// Public endpoint, but goFetch is server-only (GO_API_URL must never reach
// the browser — see lib/api.ts), so the cart page still calls it through
// this proxy rather than importing goFetch directly.
export async function POST(req: NextRequest) {
  const body = await req.text();
  const res = await goFetch("/coupons/validate", { method: "POST", body });
  const responseBody = await res.text();
  return new NextResponse(responseBody, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
