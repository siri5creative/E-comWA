import { NextRequest, NextResponse } from "next/server";
import { goFetchAsAdmin } from "@/lib/api";

export async function POST(req: NextRequest) {
  const body = await req.text();
  const res = await goFetchAsAdmin("/products", { method: "POST", body });
  const responseBody = await res.text();
  return new NextResponse(responseBody, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
