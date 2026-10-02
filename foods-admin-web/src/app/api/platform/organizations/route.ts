import {NextRequest,NextResponse} from "next/server";

export async function POST(request:NextRequest){
  const api=process.env.FOODS_API_URL??"http://localhost:8080";
  const cookie=request.headers.get("cookie")??"";
  try{
    const response=await fetch(`${api}/v1/platform/organizations`,{
      method:"POST",
      headers:{"Content-Type":"application/json",cookie},
      body:await request.text(),
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
      event:"platform_onboarding_proxy_failed",
      correlationId,
      method:"POST",
      path:"/v1/platform/organizations",
      error:error instanceof Error?error.message:"unknown_error",
    }));
    return NextResponse.json(
      {code:"api_unavailable",message:"El servicio no está disponible. Intenta nuevamente.",correlationId},
      {status:503,headers:{"X-Request-ID":correlationId}},
    );
  }
}
