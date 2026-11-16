import {AdminShell} from "@/shell/admin-shell";
import {SessionProvider, SettingsProvider} from "@/providers";
import {PayNoticeLayer} from "@/modules/payment-devices";

export const dynamic = "force-dynamic";

export default function Layout({children}:{children:React.ReactNode}){
  return <SessionProvider><SettingsProvider><AdminShell>{children}</AdminShell><PayNoticeLayer/></SettingsProvider></SessionProvider>;
}
