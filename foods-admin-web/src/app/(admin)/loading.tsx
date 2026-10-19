import {RouteSkeleton} from "@/shell/route-skeleton";

// Límite de carga dentro del shell: habilita el prefetch parcial de las rutas
// dinámicas y da respuesta inmediata al navegar entre pantallas.
export default function Loading(){return <RouteSkeleton/>}
