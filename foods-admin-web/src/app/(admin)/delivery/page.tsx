import {DeliveryPage} from "@/modules/delivery";
import {redirect} from "next/navigation";
import {pageRoutes} from "@/shared/routing/page-routes";

/** Los antiguos enlaces al tablero de delivery llevan a la lista única de Pedidos. */
export default async function Page({searchParams}:{searchParams:Promise<{tab?:string}>}){
  const params=await searchParams;
  if(params.tab==="orders")redirect(pageRoutes.orders);
  return <DeliveryPage initialTab={params.tab}/>;
}
