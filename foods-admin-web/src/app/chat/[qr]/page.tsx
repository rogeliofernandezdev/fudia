import type {Metadata,Viewport} from "next";
import {RestaurantChat} from "@/modules/concierge-web";

export const metadata:Metadata={title:"Chat del restaurante",robots:{index:false,follow:false},referrer:"no-referrer"};
// Color de la barra del navegador: el mismo gris claro de la cabecera (`--chat-panel`).
export const viewport:Viewport={themeColor:"#f0f2f8"};
export default async function ChatPage({params,searchParams}:{params:Promise<{qr:string}>;searchParams:Promise<{s?:string|string[]}>}) {
  const {qr}=await params;
  const {s}=await searchParams;
  // Reference of the QR scan: a reload repeats it, a new scan brings another.
  const scan=typeof s==="string"&&/^[A-Za-z0-9_-]{8,40}$/.test(s)?s:"";
  return <RestaurantChat qr={qr} scan={scan}/>;
}
