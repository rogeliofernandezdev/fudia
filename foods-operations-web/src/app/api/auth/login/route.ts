import {NextRequest,NextResponse} from "next/server";

export async function POST(request:NextRequest){
  const api=process.env.FOODS_API_URL??"http://localhost:8080";
  try{
    const response=await fetch(`${api}/v1/auth/login`,{
      method:"POST",
      headers:{"Content-Type":"application/json",Accept:"application/json"},
      body:await request.text(),
      cache:"no-store",
    });
    const body=await response.text();
    const headers=new Headers({"Content-Type":response.headers.get("content-type")??"application/json"});
    const setCookie=response.headers.get("set-cookie");
    if(setCookie)headers.set("Set-Cookie",setCookie);
    return new NextResponse(body,{status:response.status,headers});
  }catch{
    return NextResponse.json({code:"api_unavailable",message:"No se pudo conectar con el servicio de autenticación."},{status:503});
  }
}
