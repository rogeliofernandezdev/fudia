import "./styles/skeleton.css";
import type {CSSProperties} from "react";

/** Bloque de carga con el brillo único del sistema. Las medidas deben ser tokens. */
export function Skeleton({width,height,radius,className=""}:{width?:string;height?:string;radius?:string;className?:string}){
  const style:CSSProperties={width,height,borderRadius:radius};
  return <i className={`ds-skeleton ${className}`.trim()} style={style} aria-hidden="true"/>;
}

/**
 * Tabla de carga con las mismas columnas que la tabla real: conserva encabezados,
 * la primera celda reproduce icono + nombre + detalle y la última, las acciones.
 */
export function TableSkeleton({columns,rows=6,label="Cargando registros",lead=true,actions=true}:{columns:string[];rows?:number;label?:string;lead?:boolean;actions?:boolean}){
  return <div className="table-wrap ds-table-skeleton" role="status" aria-busy="true" aria-label={label}>
    <table aria-hidden="true">
      <thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead>
      <tbody>{Array.from({length:rows},(_,row)=><tr key={row}>{columns.map((column,index)=>{
        if(index===0&&lead)return <td key={column}><span className="ds-skeleton-lead"><Skeleton className="ds-skeleton-icon"/><span><Skeleton className="ds-skeleton-line"/><Skeleton className="ds-skeleton-line short"/></span></span></td>;
        if(index===columns.length-1&&actions)return <td key={column}><span className="ds-skeleton-actions"><Skeleton className="ds-skeleton-action"/><Skeleton className="ds-skeleton-action"/></span></td>;
        return <td key={column}><Skeleton className={`ds-skeleton-line${(row+index)%3===0?" short":""}`}/></td>;
      })}</tr>)}</tbody>
    </table>
  </div>;
}
