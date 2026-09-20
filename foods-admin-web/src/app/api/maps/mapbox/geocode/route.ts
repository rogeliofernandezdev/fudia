import { NextResponse } from "next/server";
import { z } from "zod";

const requestSchema = z.object({ address: z.string().trim().min(3).max(300) });
type MapboxFeature = {
  geometry?: { coordinates?: [number, number] };
  properties?: { full_address?: string; name?: string; place_formatted?: string };
};

export async function POST(request: Request) {
  const token = process.env.MAPBOX_ACCESS_TOKEN;
  if (!token) return NextResponse.json({ message: "Mapbox no está configurado en el servidor." }, { status: 503 });

  const parsed = requestSchema.safeParse(await request.json().catch(() => null));
  if (!parsed.success) return NextResponse.json({ message: "La dirección no es válida." }, { status: 400 });

  const params = new URLSearchParams({
    q: parsed.data.address, country: "PE", language: "es", limit: "1",
    bbox: "-81.5,-18.5,-68.5,0.5", permanent: "true", access_token: token,
  });
  const response = await fetch(`https://api.mapbox.com/search/geocode/v6/forward?${params}`, {
    cache: "no-store", signal: AbortSignal.timeout(10_000),
  });
  if (!response.ok) return NextResponse.json({ message: "No se pudo confirmar la dirección." }, { status: 502 });

  const body = await response.json() as { features?: MapboxFeature[] };
  const feature = body.features?.[0];
  const coordinates = feature?.geometry?.coordinates;
  if (!feature || !coordinates) return NextResponse.json({ message: "No se encontraron coordenadas para la dirección." }, { status: 404 });

  const normalizedAddress = feature.properties?.full_address
    ?? [feature.properties?.name, feature.properties?.place_formatted].filter(Boolean).join(", ")
    ?? parsed.data.address;
  return NextResponse.json({
    address: normalizedAddress, latitude: coordinates[1], longitude: coordinates[0],
    provider: "mapbox", permanent: true,
  });
}
