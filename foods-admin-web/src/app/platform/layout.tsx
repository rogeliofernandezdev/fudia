import {PlatformShell} from "@/modules/platform";
import {SessionProvider} from "@/providers";

export default function PlatformLayout({children}:{children:React.ReactNode}){
  return <SessionProvider><PlatformShell>{children}</PlatformShell></SessionProvider>;
}
