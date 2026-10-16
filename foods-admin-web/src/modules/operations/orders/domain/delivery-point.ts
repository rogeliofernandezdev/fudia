import type {Order} from "./types";

export type DeliveryPoint={mapUrl:string;distanceKm:number|null};

const isCoordinate=(value:unknown,limit:number):value is number=>typeof value==="number"&&Number.isFinite(value)&&Math.abs(value)<=limit;

/**
 * Punto de entrega fijado por WhatsApp en pedidos delivery de Concierge.
 * Devuelve null si el pedido no es delivery o no trae coordenadas válidas;
 * el enlace usa la URL universal de Google Maps, que abre la app en el móvil.
 */
export function deliveryPoint(order:Pick<Order,"channel"|"deliveryLatitude"|"deliveryLongitude"|"deliveryDistanceKm">):DeliveryPoint|null{
  const {channel,deliveryLatitude:lat,deliveryLongitude:lng,deliveryDistanceKm:km}=order;
  if(channel!=="delivery"||!isCoordinate(lat,90)||!isCoordinate(lng,180))return null;
  const query=encodeURIComponent(`${lat},${lng}`);
  return {
    mapUrl:`https://www.google.com/maps/search/?api=1&query=${query}`,
    distanceKm:typeof km==="number"&&Number.isFinite(km)&&km>=0?km:null,
  };
}
