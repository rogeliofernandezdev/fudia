"use client";

import dynamic from "next/dynamic";
import type { LocationMapProps } from "./maps/map-provider";
import { Skeleton } from "./skeleton";

export type { Coordinates, PlaceDetails } from "./maps/map-provider";

// El proveedor trae mapbox-gl.css y el buscador de direcciones: se descarga cuando
// un formulario muestra el mapa, no con la página que lo contiene (Clientes,
// Organización, Plataforma). Mientras tanto ocupa el mismo alto que el buscador y el lienzo.
const MapboxLocationMap = dynamic(
  () => import("./maps/mapbox-location-provider").then((module) => module.MapboxLocationMap),
  {
    ssr: false,
    loading: () => (
      <div role="status" aria-busy="true" aria-label="Cargando mapa" style={{ display: "grid", gap: "var(--space-2)" }}>
        <Skeleton height="var(--control-height)" />
        <Skeleton height="var(--size-224)" radius="var(--radius-tile)" />
      </div>
    ),
  },
);

export function LocationMap(props: LocationMapProps) {
  return <MapboxLocationMap {...props} />;
}
