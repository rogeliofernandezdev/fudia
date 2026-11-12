"use client";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {EmptyState,ErrorState,FormField,Select,Skeleton} from "@/design-system";
import {listGeographicAreas} from "../infrastructure/geography-api";
import type {GeographicArea} from "../domain/types";
import "./geographic-fields.css";

/** One catalog and hierarchy for locations and Fudia's coverage editor. */
export function GeographicFields({value,onChange,disabled=false,error}:{value:string;onChange:(code:string,area?:GeographicArea)=>void;disabled?:boolean;error?:string}){
 const[path,setPath]=useState(value);
 const selected=value||path;
 const department=selected.slice(0,2),province=selected.length>=4?selected.slice(0,4):"";
 const departments=useQuery({queryKey:["geographic-areas",""],queryFn:()=>listGeographicAreas(),staleTime:86400000});
 const provinces=useQuery({queryKey:["geographic-areas",department],queryFn:()=>listGeographicAreas(department),enabled:Boolean(department),staleTime:86400000});
 const districts=useQuery({queryKey:["geographic-areas",province],queryFn:()=>listGeographicAreas(province),enabled:Boolean(province),staleTime:86400000});
 const failed=departments.isError?departments:department&&provinces.isError?provinces:province&&districts.isError?districts:null;
 function choose(code:string,areas:GeographicArea[]){
  setPath(code);
  onChange(code.length===6?code:"",areas.find(area=>area.code===code)??departments.data?.find(area=>area.code===code)??provinces.data?.find(area=>area.code===code));
 }
 return <div className="geographic-fields span-2">
  {failed?<ErrorState title="No pudimos cargar las zonas" text={failed.error?.message} onRetry={()=>void failed.refetch()} retrying={failed.isFetching}/>:<>
   <div className="geographic-fields-grid">
    <FormField label="Departamento">{departments.isPending?<Skeleton height="var(--control-height)"/>:<Select aria-label="Departamento" value={department} disabled={disabled} onChange={event=>choose(event.target.value,departments.data??[])} required>
     <option value="">Seleccionar departamento</option>{departments.data?.map(area=><option key={area.code} value={area.code}>{area.name}</option>)}
    </Select>}</FormField>
    <FormField label="Provincia">{department&&provinces.isPending?<Skeleton height="var(--control-height)"/>:<Select aria-label="Provincia" value={province} disabled={disabled||!department} onChange={event=>choose(event.target.value||department,provinces.data??[])} required>
     <option value="">Seleccionar provincia</option>{provinces.data?.map(area=><option key={area.code} value={area.code}>{area.name}</option>)}
    </Select>}</FormField>
    <FormField label="Distrito" error={error}>{province&&districts.isPending?<Skeleton height="var(--control-height)"/>:<Select aria-label="Distrito" value={selected.length===6?selected:""} disabled={disabled||!province} onChange={event=>choose(event.target.value||province,districts.data??[])} required>
     <option value="">Seleccionar distrito</option>{districts.data?.map(area=><option key={area.code} value={area.code}>{area.name}</option>)}
    </Select>}</FormField>
   </div>
   {departments.data?.length===0&&<EmptyState icon="mapPin" title="Sin zonas disponibles" text="El catálogo geográfico aún no está disponible."/>}
   {department&&provinces.data?.length===0&&<p className="geographic-fields-empty">No hay provincias disponibles para este departamento.</p>}
   {province&&districts.data?.length===0&&<p className="geographic-fields-empty">No hay distritos disponibles para esta provincia.</p>}
  </>}
 </div>;
}
