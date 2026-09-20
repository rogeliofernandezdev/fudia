import {AdminShell} from "@/components/admin-shell";
import {SessionProvider, SettingsProvider} from "@/providers";

export const dynamic = "force-dynamic";

export default function Layout({children}:{children:React.ReactNode}){return <SessionProvider><SettingsProvider><AdminShell>{children}</AdminShell></SettingsProvider></SessionProvider>}
