"use client";
import Link from "next/link";
import {Icon} from "@/design-system/icons";
import {Logo} from "@/design-system/logo";

export default function PlatformLayout({children}:{children:React.ReactNode}){
  return <div className="platform-shell">
    <header className="platform-topbar">
      <Link href="/dashboard"><Logo/></Link>
      <span className="platform-badge"><Icon name="power" size={14}/>PLATAFORMA</span>
      <Link href="/dashboard" className="platform-exit"><Icon name="chevronLeft" size={16}/>Volver al admin</Link>
    </header>
    <main className="platform-content">{children}</main>
  </div>;
}
