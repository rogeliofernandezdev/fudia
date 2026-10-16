import {POSPage} from "@/modules/operations";

export default async function Page({searchParams}:{searchParams:Promise<{orderId?:string}>}){
 const params=await searchParams;
 return <POSPage key={params.orderId??""} initialOrderId={params.orderId??""}/>;
}
