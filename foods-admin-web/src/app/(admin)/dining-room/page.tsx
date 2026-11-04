import type {Metadata} from "next";
import {SalonManager} from "@/modules/operations";
export const metadata:Metadata={title:"Salón"};

/** `?orderId=` abre el detalle de la mesa (por ejemplo, desde un aviso). */
export default async function Page({searchParams}:{searchParams:Promise<{orderId?:string}>}){
 const params=await searchParams;
 return <SalonManager initialOrderId={params.orderId}/>;
}
