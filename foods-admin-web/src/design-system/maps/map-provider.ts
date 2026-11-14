import type { ReactNode } from "react";

export type MapProviderName = "google" | "mapbox";
export type Coordinates = { lat: number; lng: number };
/** División administrativa de la sugerencia elegida; vacía si el proveedor no la informa. */
export type PlaceDetails = { district: string; city: string };

export type LocationMapProps = {
  latitude: number | null;
  longitude: number | null;
  address: string;
  onAddressChange: (address: string) => void;
  onChange: (coordinates: Coordinates) => void;
  /** Distrito y ciudad de la sugerencia elegida en el buscador. */
  onPlaceChange?: (place: PlaceDetails) => void;
  /** Etiqueta visible del buscador; por defecto, la del local. */
  addressLabel?: ReactNode;
  /** Nombre accesible del mapa. */
  mapLabel?: string;
  /** false oculta la ayuda y la etiqueta «Ubicación definida» cuando el formulario ya confirma el punto. */
  hint?: boolean;
  /** Error de la dirección: marca el buscador y se muestra bajo él. */
  addressError?: string;
};

/** Mapa de seguimiento de un reparto: origen, destino y, si se conoce, el repartidor. */
export type RouteMapProps = {
  /** Local de donde sale el pedido. */
  origin: Coordinates | null;
  /** Punto de entrega. */
  destination: Coordinates | null;
  /** Última ubicación del repartidor; null si aún no la comparte. */
  courier: Coordinates | null;
  /** Nombre accesible del mapa. */
  label: string;
};

export function getMapProvider(): MapProviderName {
  return process.env.NEXT_PUBLIC_MAP_PROVIDER?.toLowerCase() === "google"
    ? "google"
    : "mapbox";
}

export function getMapProviderLabel(provider = getMapProvider()) {
  return provider === "google" ? "Google Maps" : "Mapbox";
}
