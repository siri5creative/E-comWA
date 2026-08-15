import { NextRequest, NextResponse } from "next/server";
import { goFetchAsAdmin } from "@/lib/api";

export async function GET(req: NextRequest) {
  const res = await goFetchAsAdmin(`/orders${req.nextUrl.search}`);
  const body = await res.text();
  return new NextResponse(body, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
