export type MapProviderName = "google" | "mapbox";
export type Coordinates = { lat: number; lng: number };

export type LocationMapProps = {
  latitude: number | null;
  longitude: number | null;
  address: string;
  onAddressChange: (address: string) => void;
  onChange: (coordinates: Coordinates) => void;
};

export function getMapProvider(): MapProviderName {
  return process.env.NEXT_PUBLIC_MAP_PROVIDER?.toLowerCase() === "google"
    ? "google"
    : "mapbox";
}

export function getMapProviderLabel(provider = getMapProvider()) {
  return provider === "google" ? "Google Maps" : "Mapbox";
}
