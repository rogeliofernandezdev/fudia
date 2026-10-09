import type { Metadata } from "next";
import { Geist_Mono, Manrope } from "next/font/google";
import "@/styles/globals.css";
import {Providers} from "@/providers/providers";
const manrope=Manrope({subsets:["latin"],variable:"--font-manrope"});
const geistMono=Geist_Mono({subsets:["latin"],variable:"--font-geist-mono"});
export const metadata:Metadata={title:"fudIA Admin",description:"Administración integral de restaurantes"};
export default function RootLayout({children}:{children:React.ReactNode}){return <html lang="es" suppressHydrationWarning><body className={`${manrope.variable} ${geistMono.variable}`}><Providers>{children}</Providers></body></html>}
