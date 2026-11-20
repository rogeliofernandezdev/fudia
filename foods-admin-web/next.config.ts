import type { NextConfig } from "next";
import {existsSync,readdirSync} from "node:fs";
import {join} from "node:path";
import {legacyPageRedirects} from "./src/shared/routing/page-routes";

// Public module APIs (`index.ts`) are pure re-export barrels. Declaring them lets
// Next load only the exports each route uses; otherwise a server page importing
// `@/modules/operations` registers every client view of the module (POS, Salón,
// Cocina, Caja…) in that route's bundle.
const moduleBarrels=readdirSync(join(process.cwd(),"src/modules"),{withFileTypes:true}).filter(entry=>entry.isDirectory()&&existsSync(join(process.cwd(),"src/modules",entry.name,"index.ts"))).map(entry=>`@/modules/${entry.name}`);

const nextConfig: NextConfig = {
  output:"standalone",
  images:{unoptimized:true},
  async headers(){
    return [{source:"/pdfjs/:path*",headers:[
      {key:"Content-Security-Policy",value:"frame-ancestors 'self'"},
      {key:"X-Frame-Options",value:"SAMEORIGIN"},
      {key:"Referrer-Policy",value:"no-referrer"},
      {key:"X-Content-Type-Options",value:"nosniff"},
    ]},
    // prepare-pdfjs versiona con ?v= el motor, el worker y los scripts y hojas del
    // visor: con la URL ligada al contenido, abrir otro PDF no los vuelve a validar.
    {source:"/pdfjs/:path*",has:[{type:"query",key:"v"}],headers:[
      {key:"Cache-Control",value:"public, max-age=31536000, immutable"},
    ]},
    // Fuentes estándar, CMaps, WASM, perfiles ICC, traducciones e iconos los pide
    // PDF.js con rutas fijas: son de la versión instalada y cambian solo al
    // actualizarla, así que basta con un día de caché.
    {source:"/pdfjs/web/:dir(standard_fonts|cmaps|wasm|iccs|locale|images)/:path*",headers:[
      {key:"Cache-Control",value:"public, max-age=86400"},
    ]}];
  },
  experimental:{
    optimizePackageImports:["@/design-system","@/providers","@/shared/management",...moduleBarrels],
    // Pages are thin server wrappers; their data lives in TanStack Query. Reusing a
    // visited page segment for 30 s makes going back to a screen instant.
    staleTimes:{dynamic:30},
  },
  async redirects(){
    return legacyPageRedirects.map(route=>({source:route.legacy,destination:route.canonical,permanent:false}));
  },
};
export default nextConfig;
