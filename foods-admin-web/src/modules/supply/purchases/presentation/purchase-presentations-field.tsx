"use client";
import {useEffect,useRef,useState} from "react";
import {createPortal} from "react-dom";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Icon,IconButton,Input,Select} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useSession} from "@/providers/session-context";
import type {PurchaseCombination,PurchasePresentationDraft} from "../domain/types";
import {listPurchaseCombinations,savePurchaseCombination} from "../infrastructure/purchases-api";

function normalizeName(text:string){return text.normalize("NFD").replace(/[\u0300-\u036f]/g,"").toLowerCase().replace(/\s+/g," ").trim()}
function duplicatesIndividual(option:PurchaseCombination,individual?:PurchaseCombination){return Boolean(individual)&&option.code!=="unit"&&normalizeName(option.name)===normalizeName(individual?.unitName??"")}
function visibleCombinations(catalog:PurchaseCombination[],row?:PurchasePresentationDraft){
 const individual=catalog.find(option=>option.code==="unit");
 // Preserve a previously selected, non-equivalent conversion with its content.
 return catalog.filter(option=>!duplicatesIndividual(option,individual)||(option.code===row?.presentationType&&Number(row.unitsPerPresentation)>0&&Number(row.unitsPerPresentation)!==1));
}

export function PurchasePresentationsField({unit,value,onChange,error,disabled,onBusyChange}:{unit:string;value:PurchasePresentationDraft[];onChange:(rows:PurchasePresentationDraft[])=>void;error?:string;disabled:boolean;onBusyChange:(busy:boolean)=>void}){
 const{organization,can}=useSession();
 const qc=useQueryClient();
 const queryKey=["purchase-combinations",organization?.id,unit];
 const query=useQuery({queryKey,queryFn:()=>listPurchaseCombinations(unit),enabled:Boolean(unit),staleTime:30000});
 const[creating,setCreating]=useState(false);
 const catalog=query.data?.items??[];
 const row=value.find(item=>item.isDefault)??value[0]??{presentationType:"",unitsPerPresentation:"",isDefault:true};
 const individual=catalog.find(option=>option.code==="unit");
 const unitName=individual?.unitName??catalog[0]?.unitName??unit;
 const options=visibleCombinations(catalog,row);
 useEffect(()=>{
  const catalog=query.data?.items??[];
  const row=value.find(item=>item.isDefault)??value[0];
  const individual=catalog.find(option=>option.code==="unit");
  const options=visibleCombinations(catalog,row);
  if(unit&&query.data&&!value.length){
   const preferred=options.find(item=>item.isDefault)??options[0];
   if(preferred)onChange([{presentationType:preferred.code,unitsPerPresentation:preferred.code==="unit"?"1":"",isDefault:true}]);
  }else if(row&&individual&&Number(row.unitsPerPresentation)===1&&catalog.some(option=>option.code===row.presentationType&&duplicatesIndividual(option,individual))){
   onChange([{presentationType:individual.code,unitsPerPresentation:"1",isDefault:true}]);
  }
 },[unit,query.data,value,onChange]);
 const selected=options.find(option=>option.code===row.presentationType);
 function close(){setCreating(false);onBusyChange(false)}
 function update(next:PurchasePresentationDraft){onChange([{...next,isDefault:true}])}
 return <div className="purchase-presentations">
  {!unit?<p className="muted">Selecciona una unidad de inventario para ver sus presentaciones de compra.</p>
   :query.isLoading?<div className="purchase-reference-loading" role="status" aria-label="Cargando presentaciones"><i/></div>
   :query.isError?<div className="purchase-validation" role="alert">{query.error.message}<Button type="button" kind="secondary" icon="refresh" disabled={disabled||query.isFetching} onClick={()=>void query.refetch()}>Reintentar</Button></div>
   :<div className="purchase-presentation-row">
     <FormField label="Presentación de compra" as="div"><div className="purchase-reference-controls">
      <Select value={row.presentationType} disabled={disabled||creating||!options.length} aria-label="Presentación de compra" onChange={event=>update({...row,presentationType:event.target.value,unitsPerPresentation:event.target.value==="unit"?"1":""})}>
       <option value="">{options.length?"Selecciona una presentación":"No hay presentaciones"}</option>
       {options.map(option=><option key={option.code} value={option.code}>{option.code==="unit"?option.unitName:duplicatesIndividual(option,individual)?`${option.name} (contenido: ${row.unitsPerPresentation})`:option.name}</option>)}
      </Select>
      {can("purchases.manage")&&<IconButton icon="plus" label="Nueva presentación" disabled={disabled||creating} onClick={()=>{setCreating(true);onBusyChange(true)}}/>}
     </div></FormField>
     <FormField label={`Contenido en ${unitName.toLowerCase()}`}><Input type="number" min="0.001" step="0.001" inputMode="decimal" value={row.unitsPerPresentation} disabled={disabled||creating||!row.presentationType||row.presentationType==="unit"} placeholder="Ej. 10" aria-label="Contenido de la presentación" onChange={event=>update({...row,unitsPerPresentation:event.target.value})}/></FormField>
     {selected&&selected.code!=="unit"&&Number(row.unitsPerPresentation)>0&&<small className="purchase-presentation-equivalence">1 {selected.name.toLowerCase()} = {row.unitsPerPresentation} {unitName.toLowerCase()}</small>}
    </div>}
  {error&&<small className="field-error" role="alert">{error}</small>}
  {creating&&createPortal(<PurchaseCombinationDialog unit={unit} unitName={unitName} close={close} save={combination=>{
   qc.setQueryData<{items:PurchaseCombination[]}>(queryKey,current=>({items:[...(current?.items??[]).filter(item=>item.code!==combination.code).map(item=>({...item,isDefault:combination.isDefault?false:item.isDefault})),combination]}));
   void qc.invalidateQueries({queryKey:["purchase-combinations",organization?.id]});
   // Select the confirmed option without guessing its article-specific factor.
   update({presentationType:combination.code,unitsPerPresentation:combination.code==="unit"?"1":"",isDefault:true});close();
  }}/>,document.body)}
 </div>;
}

export function PurchaseCombinationDialog({unit,unitName,close,save}:{unit:string;unitName:string;close:()=>void;save:(combination:PurchaseCombination)=>void}){
 const{organization}=useSession();
 const catalog=useQuery({queryKey:["purchase-combinations",organization?.id,""],queryFn:()=>listPurchaseCombinations(),staleTime:30000});
 const types=[...new Map((catalog.data?.items??[]).map(item=>[item.code,item])).values()];
 const[name,setName]=useState(""),[validation,setValidation]=useState("");
 const saving=useRef(false);
 const mutation=useMutation({mutationFn:savePurchaseCombination,retry:false,onSuccess:save,onSettled:()=>{saving.current=false}});
 function submit(){
  if(saving.current||mutation.isPending||catalog.isLoading||catalog.isError||!catalog.data)return;
  const normalizedName=name.trim().normalize("NFC");
  if(!normalizedName||[...normalizedName].length>80){setValidation("Ingresa un nombre de hasta 80 caracteres.");return}
  const individual=catalog.data.items.find(item=>item.code==="unit"&&item.unit===unit);
  if(individual&&normalizeName(normalizedName)===normalizeName(unitName)){saving.current=true;save(individual);return}
  const existing=types.find(item=>normalizeName(item.name)===normalizeName(normalizedName));
  // Reuse known types (e.g. Caja -> box); codes are never entered by the user.
  let code=existing?.code??normalizeName(normalizedName).replace(/[^a-z0-9]+/g,"-").replace(/^-|-$/g,"");
  if(!/^[a-z]/.test(code))code=`presentation-${code}`;
  code=code.slice(0,32);
  if(!existing){const stem=code;let suffix=2;while(types.some(item=>item.code===code)){const tail=`-${suffix++}`;code=stem.slice(0,32-tail.length)+tail}}
  setValidation("");saving.current=true;mutation.mutate({unit,code,name:existing?.name??normalizedName,isDefault:false});
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal supplier-modal modal-panel-in" aria-labelledby="purchase-combination-title" aria-busy={mutation.isPending}>
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="box" size={18}/></span><div><small>{unitName}</small><h2 id="purchase-combination-title">Nueva presentación</h2></div><button type="button" aria-label="Cerrar" disabled={mutation.isPending} onClick={close}><Icon name="close"/></button></header>
  <form onSubmit={event=>{event.preventDefault();event.stopPropagation()}} noValidate><div className="form-grid">
   <FormField label="Nombre" className="span-2"><Input data-dialog-initial-focus maxLength={80} value={name} disabled={mutation.isPending} onChange={event=>{setName(event.target.value);setValidation("")}}/></FormField>
   {catalog.isLoading&&<div className="purchase-reference-loading span-2" role="status" aria-label="Cargando catálogo"><i/></div>}
   {catalog.isError&&<div className="span-2"><Button type="button" kind="secondary" icon="refresh" disabled={catalog.isFetching} onClick={()=>void catalog.refetch()}>Reintentar</Button></div>}
   {(validation||mutation.isError||catalog.isError)&&<p className="field-error span-2" role="alert">{validation||mutation.error?.message||catalog.error?.message}</p>}
  </div><footer><Button type="button" kind="ghost" disabled={mutation.isPending} onClick={close}>Cancelar</Button><Button type="button" icon="check" disabled={mutation.isPending||catalog.isLoading||catalog.isError} onClick={submit}>{mutation.isPending?"Guardando…":"Guardar"}</Button></footer></form>
 </Dialog></div>;
}
