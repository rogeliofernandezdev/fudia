import {OrdersManager} from "@/modules/operations";

/** `?orderId=` abre el detalle del pedido (por ejemplo, desde un aviso). */
export default async function Page({searchParams}:{searchParams:Promise<{orderId?:string}>}){
 const params=await searchParams;
 return <OrdersManager initialOrderId={params.orderId}/>;
}
