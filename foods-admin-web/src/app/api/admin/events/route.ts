import {NextRequest,NextResponse} from "next/server";

// Canal en vivo: reenvía el flujo SSE del API sin almacenarlo ni cortarlo por
// tiempo. Es más específica que el proxy general /api/admin/[...path], que espera
// JSON y corta cada petición a los 40 s.
export const dynamic="force-dynamic";

export async function GET(request:NextRequest){
  const api=process.env.FOODS_API_URL??"http://localhost:8080";
  const headers=new Headers({Accept:"text/event-stream"});
  const cookie=request.headers.get("cookie");if(cookie)headers.set("cookie",cookie);
  try{
    // Cerrar la pestaña o cambiar de local aborta también la conexión con el API.
    const response=await fetch(`${api}/v1/admin/events`,{headers,cache:"no-store",signal:request.signal});
    if(!response.ok||!response.body){
      return new NextResponse(response.body,{status:response.status,headers:{"Content-Type":response.headers.get("content-type")??"application/json"}});
    }
    return new NextResponse(response.body,{status:200,headers:{
      "Content-Type":"text/event-stream",
      "Cache-Control":"no-cache, no-transform",
      "X-Accel-Buffering":"no",
    }});
  }catch{
    return NextResponse.json({code:"live_unavailable",message:"Las actualizaciones en vivo no están disponibles."},{status:503});
  }
}
