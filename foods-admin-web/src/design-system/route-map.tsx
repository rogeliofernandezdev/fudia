"use client";

import dynamic from "next/dynamic";
import type { RouteMapProps } from "./maps/map-provider";
import { Skeleton } from "./skeleton";

// mapbox-gl se descarga solo cuando alguien abre un seguimiento, no con la página.
const MapboxRouteMap = dynamic(
  () => import("./maps/mapbox-route-provider").then((module) => module.MapboxRouteMap),
  { ssr: false, loading: () => <Skeleton height="var(--size-240)" radius="var(--radius-tile)" /> },
);

/** Mapa de seguimiento de un reparto: local, punto de entrega, ruta y repartidor. */
export function RouteMap(props: RouteMapProps) {
  return <MapboxRouteMap {...props} />;
}
