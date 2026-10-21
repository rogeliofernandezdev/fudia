import {NextResponse} from "next/server";

export async function GET(
  _request:Request,
  {params}:{params:Promise<{qr:string}>},
){
  const{qr}=await params;
  const base=process.env.FUDIA_CONCIERGE_URL??"http://localhost:8010";
  const target=new URL(`/start/${encodeURIComponent(qr)}`,base);

  try{
    const response=await fetch(target,{
      method:"GET",
      headers:{Accept:"application/json"},
      redirect:"manual",
      cache:"no-store",
    });

    if(response.status>=300&&response.status<400){
      const location=response.headers.get("location");
      if(location)return NextResponse.redirect(location,307);
    }

    const text=await response.text();
    return new NextResponse(text,{
      status:response.status,
      headers:{"Content-Type":response.headers.get("content-type")??"application/json"},
    });
  }catch{
    return NextResponse.json(
      {code:"concierge_unavailable",message:"Fudia Concierge no está disponible en este momento."},
      {status:503},
    );
  }
}
