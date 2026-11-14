"use client";

import "mapbox-gl/dist/mapbox-gl.css";
import "../styles/route-map.css";
import { useEffect, useRef, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import type { GeoJSONSource, Map as MapboxMap, Marker as MapboxMarker } from "mapbox-gl";
import { readCssToken } from "@/design-system/theme";
import { Icon, type IconName } from "@/design-system/icons";
import type { Coordinates, RouteMapProps } from "./map-provider";

type Position = [number, number];
type MarkerKind = "origin" | "destination" | "courier";
type Pin = { marker: MapboxMarker; root: Root };

const ROUTE = "route-map-route";
const MARKER_ICONS: Record<MarkerKind, IconName> = { origin: "store", destination: "mapPin", courier: "motorbike" };
const MARKER_LABELS: Record<MarkerKind, string> = { origin: "Local", destination: "Punto de entrega", courier: "Repartidor" };
const COURIER_GLIDE_MS = 900;

const position = (point: Coordinates): Position => [point.lng, point.lat];
const key = (point: Coordinates | null) => (point ? `${point.lat.toFixed(6)},${point.lng.toFixed(6)}` : "");
const line = (coordinates: Position[]) => ({ type: "Feature" as const, properties: {}, geometry: { type: "LineString" as const, coordinates } });

/** Ruta por calles entre el local y la entrega; si Mapbox no responde queda la línea recta. */
async function drivingRoute(token: string, from: Coordinates, to: Coordinates, signal: AbortSignal): Promise<Position[]> {
  const path = `${from.lng},${from.lat};${to.lng},${to.lat}`;
  const params = new URLSearchParams({ geometries: "geojson", overview: "full", access_token: token });
  const response = await fetch(`https://api.mapbox.com/directions/v5/mapbox/driving/${path}?${params}`, { signal });
  if (!response.ok) throw new Error("route unavailable");
  const body = await response.json() as { routes?: { geometry?: { coordinates?: Position[] } }[] };
  const coordinates = body.routes?.[0]?.geometry?.coordinates;
  if (!coordinates?.length) throw new Error("route unavailable");
  return coordinates;
}

/**
 * Mapa de seguimiento: local, punto de entrega, ruta entre ambos y el
 * repartidor, que se desliza a cada ubicación nueva. No es interactivo más allá
 * de moverlo y acercarlo: no elige puntos.
 */
export function MapboxRouteMap({ origin, destination, courier, label }: RouteMapProps) {
  const container = useRef<HTMLDivElement>(null);
  const map = useRef<MapboxMap | null>(null);
  const mapbox = useRef<typeof import("mapbox-gl").default | null>(null);
  const pins = useRef<Partial<Record<MarkerKind, Pin>>>({});
  const glide = useRef(0);
  const fitted = useRef("");
  const first = origin ?? destination ?? courier;
  const hasPoint = first !== null;
  const start = useRef<Coordinates | null>(first);
  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState(false);
  const token = process.env.NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN ?? "";
  const originKey = key(origin), destinationKey = key(destination), courierKey = key(courier);

  // El primer punto conocido centra el mapa; se recuerda antes de crearlo.
  useEffect(() => { if (first) start.current = first; });

  useEffect(() => {
    if (!token || !hasPoint || !container.current || !start.current) return;
    let active = true;
    let localMap: MapboxMap | null = null;
    let resizeObserver: ResizeObserver | null = null;
    const created = pins.current;
    void import("mapbox-gl").then(({ default: mapboxgl }) => {
      if (!active || !container.current || !start.current) return;
      mapboxgl.accessToken = token;
      localMap = new mapboxgl.Map({
        container: container.current,
        style: "mapbox://styles/mapbox/streets-v12",
        center: position(start.current),
        zoom: 14,
        attributionControl: true,
      });
      localMap.addControl(new mapboxgl.NavigationControl({ showCompass: false }), "top-right");
      localMap.once("load", () => {
        if (!active || !localMap) return;
        localMap.addSource(ROUTE, { type: "geojson", data: line([]) });
        localMap.addLayer({
          id: ROUTE, type: "line", source: ROUTE,
          layout: { "line-cap": "round", "line-join": "round" },
          paint: { "line-color": readCssToken("--primary-600"), "line-width": 5, "line-opacity": 0.85 },
        });
        mapbox.current = mapboxgl;
        map.current = localMap;
        setReady(true);
      });
      resizeObserver = new ResizeObserver(() => localMap?.resize());
      resizeObserver.observe(container.current);
    }).catch(() => setFailed(true));
    return () => {
      active = false;
      window.cancelAnimationFrame(glide.current);
      resizeObserver?.disconnect();
      for (const kind of Object.keys(created) as MarkerKind[]) {
        const pin = created[kind];
        pin?.marker.remove();
        // React no permite desmontar una raíz mientras renderiza otra.
        if (pin) window.setTimeout(() => pin.root.unmount(), 0);
        delete created[kind];
      }
      localMap?.remove();
      map.current = null;
      mapbox.current = null;
      setReady(false);
    };
  }, [token, hasPoint]);

  // Marcadores fijos, ruta y encuadre. El encuadre solo cambia cuando aparece o
  // desaparece un punto: mover al repartidor no le quita el mapa de las manos al usuario.
  useEffect(() => {
    const current = map.current, mapboxgl = mapbox.current;
    if (!ready || !current || !mapboxgl) return;
    const place = (kind: MarkerKind, point: Coordinates | null) => {
      const existing = pins.current[kind];
      if (!point) {
        existing?.marker.remove();
        if (existing) window.setTimeout(() => existing.root.unmount(), 0);
        delete pins.current[kind];
        return;
      }
      if (existing) return;
      const element = document.createElement("div");
      element.className = `route-map-marker is-${kind}`;
      element.setAttribute("role", "img");
      element.setAttribute("aria-label", MARKER_LABELS[kind]);
      const root = createRoot(element);
      root.render(<Icon name={MARKER_ICONS[kind]} size={18} />);
      pins.current[kind] = { marker: new mapboxgl.Marker({ element }).setLngLat(position(point)).addTo(current), root };
    };
    place("origin", origin);
    place("destination", destination);
    place("courier", courier);
    if (origin) pins.current.origin?.marker.setLngLat(position(origin));
    if (destination) pins.current.destination?.marker.setLngLat(position(destination));

    const points = [origin, destination, courier].filter((point): point is Coordinates => point !== null);
    const frame = `${originKey}|${destinationKey}|${courier ? "courier" : ""}`;
    if (points.length && fitted.current !== frame) {
      fitted.current = frame;
      if (points.length === 1) current.easeTo({ center: position(points[0]), zoom: 15, duration: 400 });
      else {
        const bounds = new mapboxgl.LngLatBounds(position(points[0]), position(points[0]));
        points.forEach((point) => bounds.extend(position(point)));
        current.fitBounds(bounds, { padding: 56, maxZoom: 16, duration: 400 });
      }
    }

    const source = current.getSource(ROUTE) as GeoJSONSource | undefined;
    if (!source) return;
    if (!origin || !destination) { source.setData(line([])); return; }
    source.setData(line([position(origin), position(destination)]));
    const controller = new AbortController();
    drivingRoute(token, origin, destination, controller.signal)
      .then((coordinates) => source.setData(line(coordinates)))
      .catch(() => undefined);
    return () => controller.abort();
    // Las claves describen los puntos: un objeto nuevo con las mismas coordenadas no repite la ruta.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, originKey, destinationKey, Boolean(courier), token]);

  // El repartidor se desliza hasta su nueva ubicación en vez de saltar.
  useEffect(() => {
    const pin = pins.current.courier;
    if (!ready || !pin || !courier) return;
    const from = pin.marker.getLngLat();
    const to = position(courier);
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduced || (from.lng === to[0] && from.lat === to[1])) { pin.marker.setLngLat(to); return; }
    const started = performance.now();
    const step = (now: number) => {
      const progress = Math.min(1, (now - started) / COURIER_GLIDE_MS);
      pin.marker.setLngLat([from.lng + (to[0] - from.lng) * progress, from.lat + (to[1] - from.lat) * progress]);
      if (progress < 1) glide.current = window.requestAnimationFrame(step);
    };
    window.cancelAnimationFrame(glide.current);
    glide.current = window.requestAnimationFrame(step);
    return () => window.cancelAnimationFrame(glide.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, courierKey]);

  if (!token || failed || !hasPoint) return <div className="route-map-unavailable" role="status">
    <Icon name="mapPin" size={20} />
    <p>{hasPoint ? "El mapa no está disponible en este momento." : "Este pedido no tiene una ubicación en el mapa."}</p>
  </div>;

  return <div className="route-map-shell">
    <div ref={container} className="route-map-canvas" role="application" aria-label={label} />
    {!ready && <div className="route-map-loading" role="status" aria-label="Cargando mapa" />}
  </div>;
}
