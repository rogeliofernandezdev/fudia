import type {NextRequest} from "next/server";
import {proxyChat} from "@/modules/concierge-web/server";

export async function POST(request:NextRequest,{params}:{params:Promise<{qr:string;operation:string}>}) {
  const {qr,operation}=await params;
  return proxyChat(request,qr,operation);
}
