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
