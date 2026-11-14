import type {Metadata,Viewport} from "next";
import {notFound} from "next/navigation";
import {PartnerPanel} from "@/modules/delivery-panel";

// El enlace del panel es privado: no se indexa ni viaja como referencia a otros sitios (mapas).
export const metadata:Metadata={title:"Panel de reparto",robots:{index:false,follow:false},referrer:"no-referrer"};
export const viewport:Viewport={themeColor:"#052352"};
export default async function PartnerPanelPage({params}:{params:Promise<{token:string}>}) {
  const {token}=await params;
  if(!/^[a-f0-9]{32}$/.test(token))notFound();
  return <PartnerPanel token={token}/>;
}
