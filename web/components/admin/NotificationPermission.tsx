"use client";

import { useEffect } from "react";
import { requestNotificationToken } from "@/lib/firebase/client";

// Mounted once in the admin (protected) layout. Requesting permission and
// registering a token are both idempotent (the browser remembers a prior
// grant/denial; the backend upserts by token), so this is safe to run on
// every admin page load rather than tracking "first login" separately.
// Renders nothing — this is a side-effect-only component.
export function NotificationPermission() {
  useEffect(() => {
    let cancelled = false;

    (async () => {
      const token = await requestNotificationToken();
      if (!token || cancelled) return;

      try {
        await fetch("/api/admin-devices", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ fcm_device_token: token }),
        });
      } catch {
        // Best-effort — a failed registration just means this device
        // won't get push notifications until the next successful attempt.
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  return null;
}
