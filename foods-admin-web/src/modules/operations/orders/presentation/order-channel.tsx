import {Icon} from "@/design-system";
import {channelIcons,channelTone,originLabel,serviceChannel} from "./order-meta";
import "./order-channel.css";

const serviceLabels:Record<string,string>={salon:"Salón",recojo:"Recojo en local",delivery:"Delivery"};

/**
 * Canal del pedido, igual en Pedidos, Ventas y Cobros (2026-11-25): tipo de atención con su
 * icono en el color de dominio (Salón índigo, Recojo naranja, Delivery verde con moto) y, debajo,
 * el origen cuando aporta algo: «vía WhatsApp» (QR de mesa) o «Mostrador».
 */
export function OrderChannel({channel,origin,label}:{channel:string;origin?:string|null;label?:string}){
  const service=serviceChannel(channel);
  return <span className="order-channel-cell">
    <span className={`order-channel-name tone-${channelTone(service)}`}><Icon name={channelIcons[service]??"receipt"} size={16}/>{label??serviceLabels[service]??service}</span>
    {channel==="whatsapp"&&<small className="order-channel-origin"><Icon name="share" size={14}/>vía WhatsApp</small>}
    {origin==="counter"&&<small className="order-channel-origin"><Icon name="store" size={14}/>{originLabel(origin)}</small>}
  </span>;
}
