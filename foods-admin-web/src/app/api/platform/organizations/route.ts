import {NextRequest,NextResponse} from "next/server";

async function proxy(request:NextRequest){
  const api=process.env.FOODS_API_URL??"http://localhost:8080";
  const cookie=request.headers.get("cookie")??"";
  const hasBody=request.method!=="GET";
  const target=new URL(`${api}/v1/platform/organizations`);
  request.nextUrl.searchParams.forEach((value,key)=>target.searchParams.append(key,value));
  try{
    const response=await fetch(target,{
      method:request.method,
      headers:hasBody?{"Content-Type":"application/json",cookie}:{Accept:"application/json",cookie},
      body:hasBody?await request.text():undefined,
      cache:"no-store",
    });
    const text=await response.text();
    const headers=new Headers({"Content-Type":response.headers.get("content-type")??"application/json"});
    const requestId=response.headers.get("x-request-id");
    if(requestId)headers.set("X-Request-ID",requestId);
    return new NextResponse(text,{status:response.status,headers});
  }catch(error){
    const correlationId=globalThis.crypto.randomUUID();
    console.error(JSON.stringify({
      event:request.method==="GET"?"platform_organizations_proxy_failed":"platform_onboarding_proxy_failed",
      correlationId,
      method:request.method,
      path:"/v1/platform/organizations",
      error:error instanceof Error?error.message:"unknown_error",
    }));
    return NextResponse.json(
      {code:"api_unavailable",message:"El servicio no está disponible. Intenta nuevamente.",correlationId},
      {status:503,headers:{"X-Request-ID":correlationId}},
    );
  }
}

export const GET=proxy;
export const POST=proxy;
