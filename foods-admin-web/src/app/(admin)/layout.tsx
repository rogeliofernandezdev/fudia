import {AdminShell} from "@/shell/admin-shell";
import {SessionProvider, SettingsProvider} from "@/providers";
import {RestaurantOnboardingGate} from "@/modules/setup";

export const dynamic = "force-dynamic";

export default function Layout({children}:{children:React.ReactNode}){
  return <SessionProvider><SettingsProvider><AdminShell><RestaurantOnboardingGate>{children}</RestaurantOnboardingGate></AdminShell></SettingsProvider></SessionProvider>;
}
