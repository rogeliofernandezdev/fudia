import {NextRequest,NextResponse} from "next/server";

const apiUrl=()=>process.env.FOODS_API_URL??"http://localhost:8080";

export async function POST(request:NextRequest){
  const api=apiUrl();
  const cookie=request.headers.get("cookie")??"";
  if(request.cookies.get("foods_session")?.value){
    try{
      const current=await fetch(`${api}/v1/admin/context`,{headers:{Accept:"application/json",cookie},cache:"no-store"});
      if(current.ok){
        const context=await current.json() as {user?:{name?:string}};
        return NextResponse.json({code:"session_already_active",message:"Ya existe una sesión activa en este navegador.",userName:context.user?.name??"Usuario"},{status:409});
      }
      if(current.status===403)return NextResponse.json({code:"session_already_active",message:"Ya existe una sesión activa en este navegador.",userName:"Usuario"},{status:409});
      if(current.status!==401)return NextResponse.json({code:"session_validation_failed",message:"No pudimos validar la sesión actual. Intenta nuevamente."},{status:503});
    }catch{
      return NextResponse.json({code:"session_validation_failed",message:"No pudimos validar la sesión actual. Intenta nuevamente."},{status:503});
    }
  }

  const response=await fetch(`${api}/v1/auth/login`,{method:"POST",headers:{"Content-Type":"application/json"},body:await request.text(),cache:"no-store"});
  const next=new NextResponse(await response.text(),{status:response.status,headers:{"Content-Type":"application/json","Cache-Control":"no-store"}});
  const responseCookie=response.headers.get("set-cookie");
  if(responseCookie)next.headers.set("set-cookie",responseCookie);
  return next;
}

export async function DELETE(request:NextRequest){
  const cookie=request.headers.get("cookie")??"";
  try{await fetch(`${apiUrl()}/v1/auth/logout`,{method:"POST",headers:{"Content-Type":"application/json",cookie},cache:"no-store"})}catch{}
  const next=new NextResponse(null,{status:204});
  next.headers.set("set-cookie","foods_session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax");
  return next;
}
