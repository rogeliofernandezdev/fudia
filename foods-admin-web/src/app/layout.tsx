import type { Metadata, Viewport } from "next";
import { Plus_Jakarta_Sans } from "next/font/google";
import "@/styles/globals.css";
// Shared table styles are registered once, including for legacy native tables.
import "@/design-system/styles/data-table.css";
// Shared design-system styles are registered once: barrels are tree-shaken
// (`sideEffects` in package.json) and some screens reuse these classes directly.
// Heavy, feature-specific sheets (map, rich text) stay with their component.
import "@/design-system/styles/modal.css";
import "@/design-system/styles/pdf-viewer.css";
import "@/design-system/styles/pagination.css";
import "@/design-system/styles/table-actions.css";
import "@/design-system/styles/skeleton.css";
import "@/design-system/styles/tabs.css";
import "@/design-system/styles/confirm-dialog.css";
import "@/design-system/styles/form-controls.css";
import "@/design-system/styles/form-field.css";
import "@/design-system/styles/remote-modal-skeleton.css";
import "@/design-system/styles/full-screen-loader.css";
import "@/design-system/styles/icon-tile.css";
import "@/design-system/styles/stat-card.css";
import "@/design-system/styles/form-section.css";
import "@/design-system/styles/detail-view.css";
import {Providers} from "@/providers/providers";
const jakarta=Plus_Jakarta_Sans({subsets:["latin"],variable:"--font-jakarta",display:"swap"});
export const metadata:Metadata={title:"fudIA Admin",description:"Administración integral de restaurantes"};
export const viewport:Viewport={width:"device-width",initialScale:1,viewportFit:"cover",interactiveWidget:"resizes-content",themeColor:"#0d2060"};
export default function RootLayout({children}:{children:React.ReactNode}){return <html lang="es" suppressHydrationWarning><body className={jakarta.variable}><Providers>{children}</Providers></body></html>}
