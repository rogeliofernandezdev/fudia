import {NextRequest,NextResponse} from "next/server";
import {randomUUID} from "node:crypto";

// Panel de la central de Delivery Fudia: reenvía la cookie de sesión del operador
// (`fudia_partner_session`, HttpOnly) y devuelve la que fije el backend al iniciar
// o cerrar sesión. Nunca expone el token al navegador.
const SESSION_COOKIE="fudia_partner_session";
const CLEAR_COOKIE=`${SESSION_COOKIE}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0`;

async function proxy(request:NextRequest,{params}:{params:Promise<{path:string[]}>}){
  const {path}=await params;
  const api=process.env.FOODS_API_URL??"http://localhost:8080";
  const target=new URL(`${api}/v1/delivery-partner/${path.join("/")}`);
  const headers=new Headers({Accept:"application/json"});
  const session=request.cookies.get(SESSION_COOKIE)?.value;
  if(session)headers.set("cookie",`${SESSION_COOKIE}=${session}`);
  const hasBody=!["GET","HEAD"].includes(request.method);
  const logout=request.method==="POST"&&path.join("/")==="auth/logout";
  if(hasBody)headers.set("Content-Type","application/json");
  try{
    const response=await fetch(target,{method:request.method,headers,body:hasBody?await request.text():undefined,cache:"no-store",signal:AbortSignal.timeout(20000)});
    const next=response.status===204?new NextResponse(null,{status:204}):new NextResponse(await response.text(),{status:response.status,headers:{"Content-Type":response.headers.get("content-type")??"application/json"}});
    next.headers.set("Cache-Control","no-store");
    const cookie=response.headers.get("set-cookie");
    if(cookie)next.headers.set("set-cookie",cookie);
    else if((response.status===401||logout)&&session)next.headers.set("set-cookie",CLEAR_COOKIE);
    return next;
  }catch{
    const correlationId=randomUUID();
    const failed=NextResponse.json({code:"api_unavailable",message:"El servicio de reparto no está disponible. Intenta nuevamente.",correlationId},{status:503,headers:{"X-Request-ID":correlationId,"Cache-Control":"no-store"}});
    // Cerrar sesión borra la cookie aunque el backend no responda.
    if(logout&&session)failed.headers.set("set-cookie",CLEAR_COOKIE);
    return failed;
  }
}
export const GET=proxy;export const POST=proxy;
