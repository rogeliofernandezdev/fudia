"use client";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {EmptyState,ErrorState,FormField,FormSection,RowActionButton,Select,Skeleton} from "@/design-system";
import {listGeographicAreas,type GeographicArea} from "@/modules/organizations";

/** Departments and provinces filter the list; only checked districts are saved. */
export function DeliveryCoverageFields({codes,labels,onChange,disabled}:{codes:string[];labels:string[];onChange:(codes:string[],labels:string[])=>void;disabled:boolean}){
 const[department,setDepartment]=useState(codes[0]?.slice(0,2)??"");
 const[province,setProvince]=useState(codes[0]?.slice(0,4)??"");
 const departments=useQuery({queryKey:["geographic-areas",""],queryFn:()=>listGeographicAreas(),staleTime:86400000});
 const provinces=useQuery({queryKey:["geographic-areas",department],queryFn:()=>listGeographicAreas(department),enabled:Boolean(department),staleTime:86400000});
 const districts=useQuery({queryKey:["geographic-areas",province],queryFn:()=>listGeographicAreas(province),enabled:Boolean(province),staleTime:86400000});
 const failed=departments.isError?departments:department&&provinces.isError?provinces:province&&districts.isError?districts:null;
 const visible=districts.data??[];
 const allChecked=visible.length>0&&visible.every(area=>codes.includes(area.code));
 function toggle(areas:GeographicArea[],checked:boolean){
  const selected=new Map(codes.map((code,index)=>[code,labels[index]??"Distrito seleccionado"]));
  for(const area of areas){if(checked)selected.set(area.code,area.label);else selected.delete(area.code)}
  onChange([...selected.keys()],[...selected.values()]);
 }
 const other=codes.map((code,index)=>({code,label:labels[index]})).filter(area=>!province||!area.code.startsWith(province));
 return <FormSection title="Distritos de atención" icon="mapPin" bare>
  <p className="partner-coverage-help">Marca los distritos que atenderá esta central. Puedes seleccionar distritos de distintas provincias y departamentos.</p>
  {failed?<ErrorState title="No pudimos cargar los distritos" text={failed.error?.message} onRetry={()=>void failed.refetch()} retrying={failed.isFetching}/>:<>
   <div className="form-grid">
    <FormField label="Departamento">{departments.isPending?<Skeleton height="var(--control-height)"/>:<Select aria-label="Departamento" value={department} disabled={disabled} onChange={event=>{setDepartment(event.target.value);setProvince("")}}>
     <option value="">Seleccionar departamento</option>{departments.data?.map(area=><option key={area.code} value={area.code}>{area.name}</option>)}
    </Select>}</FormField>
    <FormField label="Provincia">{department&&provinces.isPending?<Skeleton height="var(--control-height)"/>:<Select aria-label="Provincia" value={province} disabled={disabled||!department} onChange={event=>setProvince(event.target.value)}>
     <option value="">Seleccionar provincia</option>{provinces.data?.map(area=><option key={area.code} value={area.code}>{area.name}</option>)}
    </Select>}</FormField>
   </div>
   {province?districts.isPending?<div role="status" aria-label="Cargando distritos"><Skeleton height="var(--control-height)"/><Skeleton height="var(--control-height)"/></div>:visible.length?<fieldset className="partner-districts">
    <legend>Distritos de la provincia</legend>
    <label className="partner-district-option partner-district-all"><input type="checkbox" checked={allChecked} disabled={disabled} onChange={event=>toggle(visible,event.target.checked)}/><span>Seleccionar todos los distritos de esta provincia</span></label>
    <div className="partner-district-grid">{visible.map(area=><label className="partner-district-option" key={area.code}><input type="checkbox" checked={codes.includes(area.code)} disabled={disabled} onChange={event=>toggle([area],event.target.checked)}/><span>{area.name}</span></label>)}</div>
   </fieldset>:<EmptyState icon="mapPin" title="Sin distritos disponibles" text="Selecciona otra provincia o vuelve a cargar el catálogo."/>:<p className="partner-coverage-help">Selecciona un departamento y una provincia para marcar sus distritos.</p>}
   {departments.data?.length===0&&<EmptyState icon="mapPin" title="Sin zonas disponibles" text="El catálogo geográfico aún no está disponible."/>}
   {department&&provinces.data?.length===0&&<p className="partner-coverage-help">No hay provincias disponibles para este departamento.</p>}
  </>}
  <p className="partner-coverage-count" aria-live="polite">{codes.length?`${codes.length} ${codes.length===1?"distrito seleccionado":"distritos seleccionados"}`:"Marca al menos un distrito para guardar la central."}</p>
  {other.length>0&&<div><p className="partner-coverage-help">Distritos seleccionados en otras provincias</p><ul className="partner-coverage-list">{other.map(area=><li key={area.code}><span>{area.label||"Distrito seleccionado"}</span><RowActionButton action="remove" label={`Quitar ${area.label||"distrito"}`} disabled={disabled} onClick={()=>toggle([{code:area.code,name:"",parentCode:"",label:area.label}],false)}/></li>)}</ul></div>}
 </FormSection>;
}
