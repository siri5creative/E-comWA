import { NextRequest, NextResponse } from "next/server";
import { goFetchAsAdmin } from "@/lib/api";

export async function POST(req: NextRequest) {
  const body = await req.text();
  const res = await goFetchAsAdmin("/admin-devices", { method: "POST", body });

  if (res.status === 204) {
    return new NextResponse(null, { status: 204 });
  }
  const responseBody = await res.text();
  return new NextResponse(responseBody, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
