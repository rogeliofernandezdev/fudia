import {PlatformShell} from "@/modules/platform";

export default function PlatformLayout({children}:{children:React.ReactNode}){
  return <PlatformShell>{children}</PlatformShell>;
}
