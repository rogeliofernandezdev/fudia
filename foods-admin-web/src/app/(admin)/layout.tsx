import {AdminShell} from "@/shell/admin-shell";
import {SessionProvider, SettingsProvider} from "@/providers";

export const dynamic = "force-dynamic";

export default function Layout({children}:{children:React.ReactNode}){return <SessionProvider><SettingsProvider><AdminShell>{children}</AdminShell></SettingsProvider></SessionProvider>}
