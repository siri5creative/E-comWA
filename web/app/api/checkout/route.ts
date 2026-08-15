import { NextRequest, NextResponse } from "next/server";
import { goFetch } from "@/lib/api";

// Proxies the client-side cart/checkout page to the Go backend, so
// GO_API_URL stays server-only (IMPLEMENTATION.md section 1: "Route
// Handler (proxy/BFF ke Go)").
export async function POST(req: NextRequest) {
  const body = await req.text();

  const res = await goFetch("/checkout", { method: "POST", body });
  const responseBody = await res.text();

  return new NextResponse(responseBody, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
