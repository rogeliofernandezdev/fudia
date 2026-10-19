import "./styles/route-skeleton.css";
import {Skeleton} from "@/design-system/skeleton";

/**
 * Esqueleto de navegación entre pantallas del shell. Se muestra al instante
 * mientras llega la nueva ruta; cada pantalla luego presenta su propio skeleton
 * de datos. Sin retraso: React mantiene el fallback ~300 ms, así que demorarlo
 * solo dejaría el área de contenido en blanco.
 */
export function RouteSkeleton(){
  return <div className="route-skeleton" role="status" aria-busy="true" aria-label="Cargando sección">
    <div className="route-skeleton-header"><Skeleton className="route-skeleton-title"/><Skeleton className="route-skeleton-subtitle"/></div>
    <div className="route-skeleton-panel">{Array.from({length:5},(_,index)=><Skeleton key={index} className="route-skeleton-row"/>)}</div>
  </div>;
}
