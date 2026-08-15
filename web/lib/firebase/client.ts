"use client";

import { initializeApp, getApps, type FirebaseApp } from "firebase/app";
import { getMessaging, getToken, isSupported } from "firebase/messaging";

let app: FirebaseApp | null = null;

function getFirebaseConfig(): Record<string, string> | null {
  const raw = process.env.NEXT_PUBLIC_FIREBASE_CONFIG;
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Record<string, string>;
  } catch {
    return null;
  }
}

function getFirebaseApp(): FirebaseApp | null {
  const config = getFirebaseConfig();
  if (!config) return null;
  if (!app) {
    app = getApps().length ? getApps()[0]! : initializeApp(config);
  }
  return app;
}

// requestNotificationToken asks the browser for notification permission
// (PRD section 6.6: "Saat admin login pertama kali di dashboard, browser
// minta izin notifikasi") and returns an FCM device token to register with
// the backend. Returns null whenever push isn't available/configured/
// granted — callers should treat that as "notifications unavailable",
// never as an error to surface to the admin.
export async function requestNotificationToken(): Promise<string | null> {
  const firebaseApp = getFirebaseApp();
  if (!firebaseApp) return null;
  if (typeof window === "undefined" || !("Notification" in window)) return null;
  if (!(await isSupported())) return null;

  const permission = await Notification.requestPermission();
  if (permission !== "granted") return null;

  const vapidKey = process.env.NEXT_PUBLIC_FIREBASE_VAPID_KEY;
  if (!vapidKey) return null;

  try {
    const messaging = getMessaging(firebaseApp);
    return await getToken(messaging, { vapidKey });
  } catch {
    return null;
  }
}
