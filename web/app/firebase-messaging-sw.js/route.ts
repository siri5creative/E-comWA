import { NextResponse } from "next/server";

// Serves the Firebase Messaging service worker at the required root path
// (/firebase-messaging-sw.js — a service worker's default scope is the
// directory it's served from) as a Route Handler instead of a static file
// in public/, so it can embed NEXT_PUBLIC_FIREBASE_CONFIG at request time.
// That value is safe to expose client-side — it's already sent to the
// browser via lib/firebase/client.ts.
export async function GET() {
  const config = process.env.NEXT_PUBLIC_FIREBASE_CONFIG ?? "{}";

  const script = `
importScripts("https://www.gstatic.com/firebasejs/12.17.1/firebase-app-compat.js");
importScripts("https://www.gstatic.com/firebasejs/12.17.1/firebase-messaging-compat.js");

firebase.initializeApp(${config});

const messaging = firebase.messaging();

// PRD section 6.6: "Notifikasi tetap muncul meski dashboard/website admin
// sedang tidak dibuka" — this handler is what makes that work; it only
// fires while the admin tab isn't focused (foreground messages are handled
// in lib/firebase/client.ts instead).
messaging.onBackgroundMessage((payload) => {
  const title = (payload.notification && payload.notification.title) || "Order Baru";
  const body = (payload.notification && payload.notification.body) || "";
  self.registration.showNotification(title, { body });
});
`;

  return new NextResponse(script, {
    headers: { "Content-Type": "application/javascript" },
  });
}
