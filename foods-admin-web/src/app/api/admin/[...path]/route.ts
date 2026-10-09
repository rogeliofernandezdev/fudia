import {NextRequest,NextResponse} from "next/server";
import {randomUUID} from "node:crypto";

async function proxy(request:NextRequest,{params}:{params:Promise<{path:string[]}>}){
 const{path}=await params;
 const api=process.env.FOODS_API_URL??"http://localhost:8080";
 const target=new URL(`${api}/v1/admin/${path.join("/")}`);
 request.nextUrl.searchParams.forEach((value,key)=>target.searchParams.append(key,value));
 const headers=new Headers({Accept:"application/json"});
 const cookie=request.headers.get("cookie");if(cookie)headers.set("cookie",cookie);
 const hasBody=!['GET','HEAD'].includes(request.method);
 if(hasBody)headers.set("Content-Type","application/json");
 try{
  // Longer than the API's 30s processing budget and 35s response deadline.
  const response=await fetch(target,{method:request.method,headers,body:hasBody?await request.text():undefined,cache:"no-store",signal:AbortSignal.timeout(40000)});
  if(response.status===204)return new NextResponse(null,{status:204});
  const next=new NextResponse(await response.text(),{status:response.status,headers:{"Content-Type":response.headers.get("content-type")??"application/json"}});
  const requestId=response.headers.get("X-Request-ID");if(requestId)next.headers.set("X-Request-ID",requestId);
  if(response.status===401)next.headers.set("set-cookie","foods_session=; Path=/; HttpOnly; Max-Age=0");
  return next;
 }catch(error){
  const correlationId=randomUUID();
  console.error("admin_proxy_unavailable",{correlationId,method:request.method,path:path.join("/"),cause:error instanceof Error?error.name:"unknown"});
  return NextResponse.json({code:"api_unavailable",message:hasBody?"No pudimos confirmar la operación. Revisa los registros antes de intentarlo nuevamente.":"El servicio no está disponible. Intenta nuevamente.",correlationId},{status:503,headers:{"X-Request-ID":correlationId}});
 }
}
export const GET=proxy;export const POST=proxy;export const PATCH=proxy;export const DELETE=proxy;
