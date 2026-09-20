import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "fudIA Operaciones",
    short_name: "fudIA",
    description: "POS, salón, cocina y caja para restaurantes.",
    start_url: "/",
    display: "standalone",
    background_color: "#F8FAFC",
    theme_color: "#176B52",
    orientation: "any",
    icons: [
      { src: "/icon.svg", sizes: "any", type: "image/svg+xml", purpose: "any" },
      { src: "/maskable-icon.svg", sizes: "any", type: "image/svg+xml", purpose: "maskable" },
    ],
  };
}
