import type {Metadata,Viewport} from "next";
import {notFound} from "next/navigation";
import {CourierPanel} from "@/modules/delivery-panel";

// El enlace del motorizado es privado: no se indexa ni viaja como referencia a otros sitios (mapas).
export const metadata:Metadata={title:"Reparto",robots:{index:false,follow:false},referrer:"no-referrer"};
export const viewport:Viewport={themeColor:"#ffffff"};
export default async function CourierPage({params}:{params:Promise<{token:string}>}) {
  const {token}=await params;
  if(!/^[a-f0-9]{32}$/.test(token))notFound();
  return <CourierPanel token={token}/>;
}
