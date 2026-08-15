import "server-only";
import { createClient } from "@/lib/supabase/server";

// Server-only helper for calling the Go backend. GO_API_URL is intentionally
// not NEXT_PUBLIC_-prefixed (IMPLEMENTATION.md section 2: "internal URL") —
// every call to it must originate from a Server Component or a Route
// Handler under app/api/, never from client-side code.

function baseUrl(): string {
  const url = process.env.GO_API_URL;
  if (!url) {
    throw new Error("GO_API_URL is not set");
  }
  return url.replace(/\/+$/, "");
}

export function goFetch(path: string, init?: RequestInit): Promise<Response> {
  return fetch(`${baseUrl()}/api/v1${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });
}

// Same as goFetch, but attaches the current admin's Supabase access token
// as a Bearer token, for endpoints protected by the Go backend's
// RequireAdmin/RequireOwner middleware. If there's no session, the request
// goes through without a token and the Go backend replies 401 itself.
export async function goFetchAsAdmin(
  path: string,
  init?: RequestInit
): Promise<Response> {
  const supabase = await createClient();
  const {
    data: { session },
  } = await supabase.auth.getSession();

  const headers = new Headers(init?.headers);
  if (session) {
    headers.set("Authorization", `Bearer ${session.access_token}`);
  }

  return goFetch(path, { ...init, headers });
}
