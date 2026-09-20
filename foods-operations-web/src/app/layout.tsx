import type { Metadata, Viewport } from "next";
import { Manrope } from "next/font/google";
import "./globals.css";
import "./navigation-state.css";
import "./loading.css";
import "./profile-menu.css";
import "./product-availability.css";

const manrope = Manrope({ subsets: ["latin"], variable: "--font-manrope" });

export const metadata: Metadata = {
  title: { default: "fudIA Operaciones", template: "%s · fudIA" },
  description: "Operación de salón, cocina y caja para restaurantes.",
  applicationName: "fudIA Operaciones",
  manifest: "/manifest.webmanifest",
  appleWebApp: { capable: true, statusBarStyle: "default", title: "fudIA" },
};

export const viewport: Viewport = {
  themeColor: "#176B52",
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="es" suppressHydrationWarning>
      <body className={manrope.variable}>{children}</body>
    </html>
  );
}
