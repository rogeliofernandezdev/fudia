import {redirect} from "next/navigation";
import {publicConciergePath} from "@/shared/routing/page-routes";

export default async function LegacyTableRedirect({params}:{params:Promise<{qr:string}>}){
  const {qr}=await params;
  redirect(publicConciergePath(qr));
}
