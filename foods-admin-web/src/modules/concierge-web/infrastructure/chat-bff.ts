import {createHmac, randomBytes, timingSafeEqual} from "node:crypto";
import {NextRequest, NextResponse} from "next/server";

const operations = new Set(["session", "state", "menu", "options", "messages"]);
const cookieName = (qr:string) => `fudia_chat_${qr}`;
const visitorName = "fudia_chat_visitor";

function sign(value:string, key:string) {
  return `${value}.${createHmac("sha256",key).update(value).digest("base64url")}`;
}
function verify(value:string|undefined, key:string):string {
  if (!value) return "";
  const [raw,mac] = value.split(".");
  if (!raw || !mac || !/^[A-Za-z0-9_-]{43}$/.test(raw)) return "";
  const expected = sign(raw,key).split(".")[1];
  return mac.length===expected.length && timingSafeEqual(Buffer.from(mac),Buffer.from(expected)) ? raw : "";
}

// Behind the public proxy `request.nextUrl` describes the internal listener
// (0.0.0.0:port), never the domain the diner opened. The browser's Origin is
// compared with the host it actually requested, as forwarded by the proxy.
function browserOrigin(request:NextRequest):URL|null {
  const origin=request.headers.get("origin");
  if (!origin || request.headers.get("sec-fetch-site")==="cross-site") return null;
  let parsed:URL;
  try { parsed=new URL(origin); } catch { return null; }
  if (parsed.protocol!=="https:" && parsed.protocol!=="http:") return null;
  const requested=(request.headers.get("x-forwarded-host")??request.headers.get("host")??request.nextUrl.host).split(",")[0].trim().toLowerCase();
  return requested && parsed.host.toLowerCase()===requested ? parsed : null;
}

export async function proxyChat(request:NextRequest, qr:string, operation:string) {
  if (!/^[a-f0-9]{32}$/.test(qr) || !operations.has(operation)) {
    return NextResponse.json({message:"Esta ruta de chat no existe."},{status:404});
  }
  // All operations are POST so capabilities never enter URLs or access logs.
  const origin=browserOrigin(request);
  if (!origin) {
    return NextResponse.json({message:"Abre el chat desde el QR del local."},{status:403});
  }
  const key=process.env.FUDIA_CONCIERGE_API_KEY;
  if (!key || key.length<32) return NextResponse.json({message:"El chat todavía no está configurado."},{status:503});
  if (Number(request.headers.get("content-length"))>12_000) return NextResponse.json({message:"El mensaje es demasiado largo."},{status:413});
  let payload:Record<string,unknown>;
  try {
    const raw=await request.text();
    if (raw.length>12_000) return NextResponse.json({message:"El mensaje es demasiado largo."},{status:413});
    payload=JSON.parse(raw);
    if (!payload || Array.isArray(payload) || typeof payload!=="object") throw new Error();
  } catch { return NextResponse.json({message:"La solicitud no es válida."},{status:400}); }
  const visitor=verify(request.cookies.get(visitorName)?.value,key) || (operation==="session"?randomBytes(32).toString("base64url"):"");
  const capability=verify(request.cookies.get(cookieName(qr))?.value,key);
  if (!visitor || (!capability && operation!=="session")) return NextResponse.json({message:"Tu sesión venció. Vuelve a abrir el chat."},{status:410});
  try {
    const upstream=await fetch(new URL(`/internal/web-chat/${operation}`,process.env.FUDIA_CONCIERGE_URL??"http://localhost:8010"),{
      method:"POST", headers:{"Content-Type":"application/json","X-Fudia-Concierge-Key":key},
      // Authentication and QR always overwrite browser-supplied fields.
      body:JSON.stringify({...payload,qr,visitor,capability}), cache:"no-store",redirect:"error",signal:AbortSignal.timeout(15_000),
    });
    const data=await upstream.json();
    const response=NextResponse.json(upstream.ok ? (operation==="session"?{ready:true}:data) : {message:typeof data.detail==="string"?data.detail:"No pudimos completar la solicitud."},{status:upstream.status,headers:{"Cache-Control":"no-store","Referrer-Policy":"no-referrer"}});
    const options={httpOnly:true,secure:origin.protocol==="https:",sameSite:"lax" as const};
    if (operation==="session" && upstream.ok && typeof data.capability==="string") {
      response.cookies.set(visitorName,sign(visitor,key),{...options,path:"/api/public/chat",maxAge:180*24*3600});
      response.cookies.set(cookieName(qr),sign(data.capability,key),{...options,path:`/api/public/chat/${qr}`,maxAge:Math.min(Number(data.expiresIn)||43_200,43_200)});
    }
    if (upstream.status===410) response.cookies.set(cookieName(qr),"",{...options,path:`/api/public/chat/${qr}`,maxAge:0});
    return response;
  } catch {
    return NextResponse.json({message:"No pudimos conectar con el chat. Inténtalo de nuevo."},{status:503});
  }
}
