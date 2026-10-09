"use client";
import {useRef,useState} from "react";
import {createPortal} from "react-dom";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Icon,IconButton,Input,Select} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useSession} from "@/providers/session-context";
import {createInventoryUnit,listInventoryUnits} from "../infrastructure/inventory-units-api";
import type {InventoryUnit,InventoryUnitDraft} from "../domain/inventory-unit";

export function PurchaseUnitField({value,onChange,error,disabled,onBusyChange}:{value:string;onChange:(value:string)=>void;error?:string;disabled:boolean;onBusyChange:(busy:boolean)=>void}){
 const{organization,can}=useSession();
 const qc=useQueryClient();
 const queryKey=["inventory-units",organization?.id];
 const query=useQuery({queryKey,queryFn:listInventoryUnits,staleTime:30000});
 const[creating,setCreating]=useState(false);
 const[selected,setSelected]=useState<InventoryUnit|null>(null);
 const units=query.data?.items??[];
 const options=selected&&!units.some(unit=>unit.code===selected.code)?[...units,selected]:units;
 function close(){setCreating(false);onBusyChange(false)}
 return <>
  <FormField label="Unidad de inventario" as="div" error={query.isError?query.error.message:error}>
   {query.isLoading?<div className="purchase-reference-loading" role="status" aria-label="Cargando unidades"><i/></div>:<div className="purchase-reference-controls">
    <Select aria-label="Unidad de inventario" value={value} disabled={disabled||creating||query.isError||options.length===0} aria-invalid={Boolean(error)||query.isError} onChange={event=>onChange(event.target.value)}>
     <option value="">{query.isError?"No pudimos cargar las unidades":options.length?"Selecciona una unidad":"No hay unidades registradas"}</option>
     {options.map(unit=><option key={unit.id} value={unit.code}>{unit.name}</option>)}
    </Select>
    {query.isError?<Button type="button" kind="secondary" icon="refresh" disabled={disabled||query.isFetching} onClick={()=>void query.refetch()}>Reintentar</Button>:can("purchases.manage")&&<IconButton icon="plus" label="Nueva unidad" disabled={disabled||creating} onClick={()=>{setCreating(true);onBusyChange(true)}}/>}
   </div>}
  </FormField>
  {creating&&createPortal(<InventoryUnitDialog close={close} save={unit=>{setSelected(unit);onChange(unit.code);qc.setQueryData<{items:InventoryUnit[]}>(queryKey,current=>({items:[...(current?.items??[]).filter(item=>item.code!==unit.code),unit]}));void qc.invalidateQueries({queryKey});close()}}/>,document.body)}
 </>;
}

export function InventoryUnitDialog({close,save}:{close:()=>void;save:(unit:InventoryUnit)=>void}){
 const[draft,setDraft]=useState<InventoryUnitDraft>({code:"",name:""});
 const[validation,setValidation]=useState("");
 const saving=useRef(false);
 const mutation=useMutation({mutationFn:createInventoryUnit,retry:false,onSuccess:save,onSettled:()=>{saving.current=false}});
 function submit(){
  if(saving.current)return;
  const code=draft.code.trim().toLowerCase(),name=draft.name.trim();
  if(!/^[a-z][a-z0-9_-]{0,31}$/.test(code)||!name||[...name].length>80){setValidation("Ingresa un nombre y un código de hasta 32 caracteres, sin espacios.");return}
  setValidation("");saving.current=true;mutation.mutate({code,name});
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal supplier-modal modal-panel-in" aria-labelledby="inventory-unit-title" aria-busy={mutation.isPending}>
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="box" size={18}/></span><div><small>NUEVA UNIDAD</small><h2 id="inventory-unit-title">Unidad de inventario</h2></div><button type="button" aria-label="Cerrar" disabled={mutation.isPending} onClick={close}><Icon name="close"/></button></header>
  <form onSubmit={event=>{event.preventDefault();event.stopPropagation()}} noValidate><div className="form-grid">
   <FormField label="Nombre"><Input data-dialog-initial-focus value={draft.name} maxLength={80} disabled={mutation.isPending} onChange={event=>setDraft({...draft,name:event.target.value})}/></FormField>
   <FormField label="Código"><Input value={draft.code} maxLength={32} disabled={mutation.isPending} onChange={event=>setDraft({...draft,code:event.target.value})}/></FormField>
   {(validation||mutation.isError)&&<p className="field-error span-2" role="alert">{validation||mutation.error?.message}</p>}
  </div>
  <footer><Button type="button" kind="ghost" disabled={mutation.isPending} onClick={close}>Cancelar</Button><Button type="button" disabled={mutation.isPending} onClick={submit}>{mutation.isPending?"Guardando…":"Guardar"}</Button></footer></form>
 </Dialog></div>;
}
