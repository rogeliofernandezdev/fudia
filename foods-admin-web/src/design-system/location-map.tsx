"use client";

import { MapboxLocationMap } from "./maps/mapbox-location-provider";
import type { LocationMapProps } from "./maps/map-provider";

export type { Coordinates } from "./maps/map-provider";

export function LocationMap(props: LocationMapProps) {
  return <MapboxLocationMap {...props} />;
}
