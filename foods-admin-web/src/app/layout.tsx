import type { Metadata, Viewport } from "next";
import { Manrope } from "next/font/google";
import "@/styles/globals.css";
// Shared table styles are registered once, including for legacy native tables.
import "@/design-system/styles/data-table.css";
import {Providers} from "@/providers/providers";
const manrope=Manrope({subsets:["latin"],variable:"--font-manrope",display:"swap"});
export const metadata:Metadata={title:"fudIA Admin",description:"Administración integral de restaurantes"};
export const viewport:Viewport={width:"device-width",initialScale:1,viewportFit:"cover",interactiveWidget:"resizes-content",themeColor:"#1a2151"};
export default function RootLayout({children}:{children:React.ReactNode}){return <html lang="es" suppressHydrationWarning><body className={manrope.variable}><Providers>{children}</Providers></body></html>}
