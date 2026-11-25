import type {Metadata,Viewport} from "next";
import {cookies} from "next/headers";
import {PartnerLogin} from "@/modules/delivery-panel";

export const metadata:Metadata={title:"Panel de reparto",robots:{index:false,follow:false}};
export const viewport:Viewport={themeColor:"#ffffff"};
export default async function PartnerLoginPage(){
  // Sin cookie no hay sesión que validar: el formulario aparece de inmediato.
  const hasSession=Boolean((await cookies()).get("fudia_partner_session")?.value);
  return <PartnerLogin hasSession={hasSession}/>;
}
