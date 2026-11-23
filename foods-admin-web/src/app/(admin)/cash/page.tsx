import {CashPage} from "@/modules/operations";
export default async function Page({searchParams}:{searchParams:Promise<{tab?:string}>}){
 const {tab}=await searchParams;
 const initialTab=tab==="pagos"||tab==="history"||tab==="gastos"?tab:"registers";
 return <CashPage key={initialTab} initialTab={initialTab}/>;
}
