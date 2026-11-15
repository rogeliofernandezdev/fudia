import type {Metadata,Viewport} from "next";
import {cookies} from "next/headers";
import {redirect} from "next/navigation";
import {PartnerPanel} from "@/modules/delivery-panel";

// Panel de la central de Delivery Fudia: exige la sesión de un operador.
export const metadata:Metadata={title:"Panel de reparto",robots:{index:false,follow:false},referrer:"no-referrer"};
export const viewport:Viewport={themeColor:"#052352"};
export default async function PartnerPanelPage(){
  if(!(await cookies()).get("fudia_partner_session")?.value)redirect("/delivery-panel/login");
  return <PartnerPanel/>;
}
