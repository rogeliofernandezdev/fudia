import {DeliveryPage} from "@/modules/delivery";

/** `?tab=` abre una pestaña: orders, providers, fees, failed, apps o couriers. */
export default async function Page({searchParams}:{searchParams:Promise<{tab?:string}>}){
  const params=await searchParams;
  return <DeliveryPage initialTab={params.tab}/>;
}
