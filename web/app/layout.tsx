import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  weight: ["400", "500", "700", "900"],
});

export const metadata: Metadata = {
  title: "Toko Online",
  description: "Belanja online, konfirmasi cepat lewat WhatsApp.",
};

// Bare HTML shell only — the public storefront chrome (Navbar/Footer/cart)
// lives in app/(public)/layout.tsx and the admin chrome in
// app/admin/layout.tsx, since they must never mix.
export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="id" className={`${inter.variable} h-full antialiased`}>
      <body className="min-h-full flex flex-col font-sans">{children}</body>
    </html>
  );
}
