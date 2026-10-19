import type { Metadata, Viewport } from "next";
import { Manrope } from "next/font/google";
import "@/styles/globals.css";
// Shared table styles are registered once, including for legacy native tables.
import "@/design-system/styles/data-table.css";
// Shared design-system styles are registered once: barrels are tree-shaken
// (`sideEffects` in package.json) and some screens reuse these classes directly.
// Heavy, feature-specific sheets (map, rich text) stay with their component.
import "@/design-system/styles/modal.css";
import "@/design-system/styles/pagination.css";
import "@/design-system/styles/table-actions.css";
import "@/design-system/styles/skeleton.css";
import "@/design-system/styles/tabs.css";
import "@/design-system/styles/confirm-dialog.css";
import "@/design-system/styles/form-controls.css";
import "@/design-system/styles/form-field.css";
import "@/design-system/styles/remote-modal-skeleton.css";
import "@/design-system/styles/full-screen-loader.css";
import {Providers} from "@/providers/providers";
const manrope=Manrope({subsets:["latin"],variable:"--font-manrope",display:"swap"});
export const metadata:Metadata={title:"fudIA Admin",description:"Administración integral de restaurantes"};
export const viewport:Viewport={width:"device-width",initialScale:1,viewportFit:"cover",interactiveWidget:"resizes-content",themeColor:"#1a2151"};
export default function RootLayout({children}:{children:React.ReactNode}){return <html lang="es" suppressHydrationWarning><body className={manrope.variable}><Providers>{children}</Providers></body></html>}
