import type { NextConfig } from "next";
import {legacyPageRedirects} from "./src/shared/routing/page-routes";

const nextConfig: NextConfig = {
  output:"standalone",
  images:{unoptimized:true},
  async redirects(){
    return legacyPageRedirects.map(route=>({source:route.legacy,destination:route.canonical,permanent:false}));
  },
};
export default nextConfig;
