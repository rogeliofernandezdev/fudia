"use client";

import "mapbox-gl/dist/mapbox-gl.css";
import "../styles/location-map.css";
import { useEffect, useRef, useState } from "react";
import type { Map as MapboxMap, Marker as MapboxMarker } from "mapbox-gl";
import { Icon } from "@/design-system/icons";
import { Input } from "@/design-system/page-header";
import type { Coordinates, LocationMapProps } from "./map-provider";

type Suggestion = { mapbox_id: string; name: string; place_formatted?: string };
type SearchFeature = {
  geometry: { coordinates: [number, number] };
  properties?: { full_address?: string; name?: string; place_formatted?: string };
};

const DEFAULT_CENTER: [number, number] = [-77.042793, -12.046374];
const PERU_BOUNDS: [[number, number], [number, number]] = [
  [-81.5, -18.5],
  [-68.5, 0.5],
];
const PERU_SEARCH_BBOX = "-81.5,-18.5,-68.5,0.5";

function createSessionToken() {
  if (typeof globalThis.crypto?.randomUUID === "function") return globalThis.crypto.randomUUID();
  const bytes = new Uint8Array(16);
  globalThis.crypto?.getRandomValues?.(bytes);
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (value) => value.toString(16).padStart(2, "0"));
  return `${hex.slice(0, 4).join("")}-${hex.slice(4, 6).join("")}-${hex.slice(6, 8).join("")}-${hex.slice(8, 10).join("")}-${hex.slice(10).join("")}`;
}

export function MapboxLocationMap({
  latitude, longitude, address, onAddressChange, onChange,
}: LocationMapProps) {
  const container = useRef<HTMLDivElement>(null);
  const map = useRef<MapboxMap | null>(null);
  const marker = useRef<MapboxMarker | null>(null);
  const sessionToken = useRef("");
  const searchSequence = useRef(0);
  const suppressNextSearch = useRef(false);
  const searchTriggeredByTyping = useRef(false);
  const callbacks = useRef({ onChange });
  const initialPosition = useRef<[number, number]>(
    longitude != null && latitude != null ? [longitude, latitude] : DEFAULT_CENTER,
  );
  const initialZoom = useRef(latitude != null && longitude != null ? 17 : 11);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [searching, setSearching] = useState(false);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const token = process.env.NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN ?? "";

  useEffect(() => {
    sessionToken.current = createSessionToken();
  }, []);

  useEffect(() => {
    callbacks.current = { onChange };
  }, [onChange]);

  useEffect(() => {
    if (!token || !container.current) return;
    let active = true;
    let localMap: MapboxMap | null = null;
    let localMarker: MapboxMarker | null = null;
    void import("mapbox-gl").then(({ default: mapboxgl }) => {
      if (!active || !container.current) return;
      mapboxgl.accessToken = token;
      const initial = initialPosition.current;
      localMap = new mapboxgl.Map({
        container: container.current,
        style: "mapbox://styles/mapbox/streets-v12",
        center: initial,
        zoom: initialZoom.current,
        maxBounds: PERU_BOUNDS,
        attributionControl: true,
      });
      localMap.addControl(new mapboxgl.NavigationControl({ showCompass: false }), "top-right");
      localMarker = new mapboxgl.Marker({ color: "#2f5bc7", draggable: true })
        .setLngLat(initial)
        .addTo(localMap);
      localMarker.on("dragend", () => {
        const point = localMarker!.getLngLat();
        callbacks.current.onChange({ lat: point.lat, lng: point.lng });
      });
      localMap.on("click", (event) => {
        localMarker!.setLngLat(event.lngLat);
        callbacks.current.onChange({ lat: event.lngLat.lat, lng: event.lngLat.lng });
      });
      map.current = localMap;
      marker.current = localMarker;
    }).catch(() => setError("No se pudo cargar Mapbox."));
    return () => {
      active = false;
      localMarker?.remove();
      localMap?.remove();
      marker.current = null;
      map.current = null;
    };
  }, [token]);

  useEffect(() => {
    if (latitude == null || longitude == null || !map.current || !marker.current) return;
    marker.current.setLngLat([longitude, latitude]);
  }, [latitude, longitude]);

  useEffect(() => {
    const query = address.trim();
    if (suppressNextSearch.current) {
      suppressNextSearch.current = false;
      searchTriggeredByTyping.current = false;
      searchSequence.current += 1;
      setSuggestions([]);
      setOpen(false);
      setSearching(false);
      return;
    }
    if (!searchTriggeredByTyping.current) {
      setSuggestions([]);
      setOpen(false);
      setSearching(false);
      return;
    }
    if (!token || query.length < 5 || !sessionToken.current) {
      setSuggestions([]);
      setOpen(false);
      setSearching(false);
      return;
    }
    const sequence = ++searchSequence.current;
    const controller = new AbortController();
    setSearching(true);
    const timer = window.setTimeout(async () => {
      try {
        const params = new URLSearchParams({
          q: query,
          access_token: token,
          session_token: sessionToken.current,
          country: "PE",
          language: "es",
          bbox: PERU_SEARCH_BBOX,
          limit: "6",
        });
        const response = await fetch(`https://api.mapbox.com/search/searchbox/v1/suggest?${params}`, {
          signal: controller.signal,
        });
        if (!response.ok) throw new Error();
        const result = await response.json() as { suggestions?: Suggestion[] };
        if (sequence !== searchSequence.current) return;
        setSuggestions(result.suggestions ?? []);
        setOpen(true);
        setError("");
      } catch (reason) {
        if (reason instanceof DOMException && reason.name === "AbortError") return;
        if (sequence === searchSequence.current) setError("No se pudieron consultar direcciones en Mapbox.");
      } finally {
        if (sequence === searchSequence.current) setSearching(false);
      }
    }, 400);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [address, token]);

  async function selectAddress(suggestion: Suggestion) {
    searchSequence.current += 1;
    searchTriggeredByTyping.current = false;
    setSuggestions([]);
    setOpen(false);
    setSearching(true);
    try {
      const retrieveParams = new URLSearchParams({
        access_token: token,
        session_token: sessionToken.current,
      });
      const retrieveResponse = await fetch(
        `https://api.mapbox.com/search/searchbox/v1/retrieve/${encodeURIComponent(suggestion.mapbox_id)}?${retrieveParams}`,
      );
      if (!retrieveResponse.ok) throw new Error();
      const retrieve = await retrieveResponse.json() as { features?: SearchFeature[] };
      const feature = retrieve.features?.[0];
      if (!feature) throw new Error();
      const selectedAddress = feature.properties?.full_address
        ?? [suggestion.name, suggestion.place_formatted].filter(Boolean).join(", ");
      const [temporaryLongitude, temporaryLatitude] = feature.geometry.coordinates;

      suppressNextSearch.current = true;
      onAddressChange(selectedAddress);
      map.current?.flyTo({
        center: [temporaryLongitude, temporaryLatitude],
        zoom: 17,
        essential: true,
      });
      marker.current?.setLngLat([temporaryLongitude, temporaryLatitude]);
      setSearching(false);

      const permanentResponse = await fetch("/api/maps/mapbox/geocode", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ address: selectedAddress }),
      });
      if (!permanentResponse.ok) throw new Error();
      const permanent = await permanentResponse.json() as {
        address: string; latitude: number; longitude: number;
      };
      const position: Coordinates = { lat: permanent.latitude, lng: permanent.longitude };
      if (permanent.address && permanent.address !== selectedAddress) {
        suppressNextSearch.current = true;
        onAddressChange(permanent.address);
      }
      onChange(position);
      const coordinateChanged = Math.abs(position.lat - temporaryLatitude) > 0.00001
        || Math.abs(position.lng - temporaryLongitude) > 0.00001;
      if (coordinateChanged) {
        map.current?.flyTo({ center: [position.lng, position.lat], zoom: 17, essential: true });
        marker.current?.setLngLat([position.lng, position.lat]);
      }
      setError("");
      sessionToken.current = createSessionToken();
    } catch {
      setError("No se pudo confirmar permanentemente esta dirección.");
    } finally {
      setSearching(false);
    }
  }

  if (!token) return <MapSetupMessage />;

  return <div className="location-map-wrapper">
    <label className="location-map-label">
      Dirección del local
      <span className="location-map-search">
        <span className="location-map-search-icon"><Icon name="search" size={14}/></span>
        <Input value={address} onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
          searchTriggeredByTyping.current = true;
          onAddressChange(event.target.value);
        }}
          onBlur={() => window.setTimeout(() => setOpen(false), 150)}
          placeholder="Escribe calle, número y distrito" autoComplete="off" role="combobox"
          aria-autocomplete="list" aria-expanded={open} className="location-map-input" />
        {searching && <span className="location-map-spinner"><Icon name="refresh" size={14}/></span>}
        {open && suggestions.length > 0 && <div role="listbox" className="location-map-suggestions">
          {suggestions.map((suggestion) => <button key={suggestion.mapbox_id} type="button"
            role="option" aria-selected="false" onMouseDown={(event) => event.preventDefault()}
            onClick={() => void selectAddress(suggestion)}
            className="location-map-suggestion">
            <span className="location-map-pin"><Icon name="grid" size={14}/></span>
            <span className="location-map-suggestion-text"><strong>{suggestion.name}</strong>
              {suggestion.place_formatted && <span>{suggestion.place_formatted}</span>}
            </span>
          </button>)}
          <p className="location-map-suggestions-footer">Resultados de Mapbox</p>
        </div>}
      </span>
    </label>
    <div ref={container} className="location-map-canvas" aria-label="Mapa para seleccionar la ubicación del local" />
    <div className="location-map-footer">
      <p>Selecciona una sugerencia y, si hace falta, ajusta el punto en el mapa.</p>
      {latitude != null && longitude != null && <span className="location-map-badge">Ubicación definida</span>}
    </div>
    {error && <p role="alert" className="location-map-error">{error}</p>}
  </div>;
}

function MapSetupMessage() {
  return <div className="location-map-setup">
    <div><span className="location-map-setup-icon"><Icon name="grid" size={24}/></span>
      <p className="location-map-setup-title">Mapa listo para configurar</p>
      <p className="location-map-setup-desc">Configura el token público de Mapbox.</p>
    </div>
  </div>;
}
