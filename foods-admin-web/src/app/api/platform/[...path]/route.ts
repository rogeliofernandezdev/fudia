import {NextRequest,NextResponse} from "next/server";

async function proxy(request:NextRequest,{params}:{params:Promise<{path:string[]}>}){
  const{path}=await params;
  const api=process.env.FOODS_API_URL??"http://localhost:8080";
  const target=new URL(`${api}/v1/platform/${path.join("/")}`);
  request.nextUrl.searchParams.forEach((value,key)=>target.searchParams.append(key,value));
  const headers=new Headers({Accept:"application/json"});
  const cookie=request.headers.get("cookie");
  if(cookie)headers.set("cookie",cookie);
  const hasBody=!["GET","HEAD"].includes(request.method);
  if(hasBody)headers.set("Content-Type","application/json");
  try{
    const response=await fetch(target,{method:request.method,headers,body:hasBody?await request.text():undefined,cache:"no-store"});
    const responseHeaders=new Headers();
    const contentType=response.headers.get("content-type");
    const requestId=response.headers.get("x-request-id");
    if(contentType)responseHeaders.set("Content-Type",contentType);
    if(requestId)responseHeaders.set("X-Request-ID",requestId);
    if(response.status===204)return new NextResponse(null,{status:204,headers:responseHeaders});
    const text=await response.text();
    return new NextResponse(text,{status:response.status,headers:responseHeaders});
  }catch(error){
    const correlationId=globalThis.crypto.randomUUID();
    console.error(JSON.stringify({event:"platform_proxy_failed",correlationId,method:request.method,path:`/v1/platform/${path.join("/")}`,error:error instanceof Error?error.message:"unknown_error"}));
    return NextResponse.json({code:"api_unavailable",message:"El servicio no está disponible. Intenta nuevamente.",correlationId},{status:503,headers:{"X-Request-ID":correlationId}});
  }
}

export const GET=proxy;
export const POST=proxy;
export const PATCH=proxy;
