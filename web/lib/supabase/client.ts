import { createBrowserClient } from "@supabase/ssr";

// Browser-side Supabase client — used only by the admin login form to call
// supabase.auth.signInWithPassword(). No other client code should read
// data directly from Supabase; everything else goes through the Go API.
export function createClient() {
  return createBrowserClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!
  );
}
