"use client";
/* eslint-disable @next/next/no-img-element -- Product images use tenant-configured URLs outside Next image domains. */
import "./product-availability.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,SearchField,Select,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSession} from "@/providers/session-context";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {AvailabilityChange,AvailabilityItem,AvailabilityReasonDraft,AvailabilityStatus} from "../domain/types";
import {listAvailability,listAvailabilityCategories,updateAvailability} from "../infrastructure/availability-api";
import {AvailabilityReasonDialog} from "./availability-reason-dialog";
import {AvailabilityHistoryDialog} from "./availability-history-dialog";

const labels:Record<AvailabilityStatus,string>={available:"Disponible",low:"Pocas unidades",sold_out:"Agotado",unavailable:"Fuera de horario"};
const tones:Record<AvailabilityStatus,"green"|"orange"|"gray">={available:"green",low:"orange",sold_out:"gray",unavailable:"gray"};

export function ProductAvailabilityManager(){
 const client=useQueryClient();
 const{notify}=useFeedback();
 const{organization,location,can,isLoading:sessionLoading}=useSession();
 const[search,setSearch]=useState("");
 const[categoryId,setCategoryId]=useState("");
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(10);
 const[portions,setPortions]=useState<Record<string,string>>({});
 const[change,setChange]=useState<AvailabilityChange|null>(null);
 const[history,setHistory]=useState<AvailabilityItem|null>(null);
 const canRead=can("menu.read");
 const canManage=can("menu.manage");

 const query=useQuery({
  queryKey:["product-availability",organization?.id,location?.id,search,categoryId,page,size],
  queryFn:()=>listAvailability({search,categoryId,page,pageSize:size}),
  enabled:!!location?.id&&canRead,
 });
 const categories=useQuery({queryKey:["categories","availability-filter",organization?.id],queryFn:listAvailabilityCategories,enabled:canRead});
 const update=useMutation({
  mutationFn:({item,status,portionQuantity,reasonCode,reason}:AvailabilityChange&AvailabilityReasonDraft)=>
   updateAvailability(item.productId,{status,portionQuantity,note:item.note,reasonCode,reason}),
  retry:false,
  onSuccess:(_,variables)=>{
   setChange(null);
   if(variables.kind==="quota"){
    setPortions(value=>{const next={...value};delete next[variables.item.productId];return next});
   }
   void client.invalidateQueries({queryKey:["product-availability"]});
   void client.invalidateQueries({queryKey:["products"]});
   void client.invalidateQueries({queryKey:["availability-history"]});
   void client.invalidateQueries({queryKey:["order-catalog"]});
   void client.invalidateQueries({queryKey:["dashboard"]});
   void client.invalidateQueries({queryKey:["combos"]});
   notify({
    tone:"success",
    title:variables.kind==="quota"?"Cupo actualizado":"Disponibilidad actualizada",
    message:variables.kind==="quota"?"El cupo del día quedó guardado.":"El estado manual ya aplica al local activo.",
   });
  },
  onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message}),
 });

 function saveQuota(item:AvailabilityItem){
  const portionQuantity=Number(portions[item.productId]??item.portionQuantity??0);
  const minimum=Math.max(1,item.soldQuantity);
  if(!canManage||update.isPending||!Number.isInteger(portionQuantity)||portionQuantity<minimum){
   return;
  }
  setChange({item,status:item.manualStatus,portionQuantity,kind:"quota"});
 }

 function setManualStatus(item:AvailabilityItem,status:"available"|"sold_out"){
  if(!canManage||update.isPending)return;
  // Reactivar es la acción normal y no pide motivo: se guarda en un toque.
  if(status==="available"){void update.mutateAsync({item,status,portionQuantity:null,kind:"status",reasonCode:"",reason:""}).catch(()=>{/* onError comunica la respuesta del API. */});return;}
  setChange({item,status,portionQuantity:null,kind:"status"});
 }

 const items=query.data?.items??[];
 const businessDate=query.data?.businessDate
  ?formatRegionalCalendarDate(query.data.businessDate,location?.country,{weekday:"short",day:"2-digit",month:"short"})
  :"Cargando fecha…";

 return <>
  <div className="availability-page-heading"><PageHeader title="Disponibilidad" description="Administra el cupo diario y consulta la disponibilidad real de cada producto en el local activo." action={<span className="availability-date"><Icon name="calendar" size={18}/>{businessDate}</span>}/></div>
  <section className="panel availability-panel">
   <header className="availability-toolbar">
    <SearchField value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar plato o producto…"/>
    <Select chip value={categoryId} onChange={event=>{setCategoryId(event.target.value);setPage(1)}} disabled={categories.isLoading} aria-label="Filtrar por categoría">
     <option value="">Todas las categorías</option>
     {categories.data?.items.map(category=><option value={category.id} key={category.id}>{category.name}</option>)}
    </Select>
    {canRead&&!canManage&&<span className="availability-read-only"><Icon name="lock" size={16}/>Solo lectura</span>}
   </header>

   {sessionLoading?<div className="availability-table-wrap hover-scroll" tabIndex={0}><AvailabilitySkeleton/></div>:!canRead?
    <div className="availability-state"><Icon name="alert" size={24}/><b>Sin permiso de lectura</b><p>Tu rol no permite consultar la disponibilidad.</p></div>
   :query.isLoading?<div className="availability-table-wrap hover-scroll" tabIndex={0}><AvailabilitySkeleton/></div>:query.isError?
    <div className="availability-state"><Icon name="alert" size={24}/><b>No pudimos cargar la carta</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="availability-state"><Icon name="availability" size={24}/><b>Sin productos</b><p>No hay productos activos que coincidan con los filtros.</p></div>
   :<div className="availability-table-wrap hover-scroll" tabIndex={0}>
    <div className="availability-list-head" aria-hidden="true"><span><Icon name="dish" size={16}/>Producto</span><span><Icon name="availability" size={16}/>Disponibilidad</span><span><Icon name="cookingPot" size={16}/>Cupo de hoy</span><span><Icon name="gauge" size={16}/>Unidades de hoy</span><span>Acciones</span></div>
    <div className="availability-grid">
     {items.map(item=>{
      const derived=item.source==="schedule"||item.source==="combo_components";
      const pending=update.isPending&&update.variables?.item.productId===item.productId;
      const portionValue=portions[item.productId]??String(item.portionQuantity??0);
      const portionChanged=item.quantityControl==="portions"&&portionValue!==String(item.portionQuantity??0);
      const portionMinimum=Math.max(1,item.soldQuantity);
      const portionNumber=Number(portionValue);
      const portionValid=Number.isInteger(portionNumber)&&portionNumber>=portionMinimum;
      const manuallySoldOut=item.manualStatus==="sold_out";
      const quantityExhausted=(item.source==="portions"||item.source==="inventory")&&item.status==="sold_out";
      const remainingAmount=item.remaining===null?null:formatRegionalNumber(Number(item.remaining),location?.country,{maximumFractionDigits:item.quantityControl==="inventory"?3:0});
      const statusCaption=derived?"Calculado automáticamente":manuallySoldOut?"Agotado manualmente":"";
      const notice=derived?(item.source==="schedule"?"Revisa el horario":"Falta un componente")
       :item.quantityControl==="portions"&&item.source==="portions"&&item.status==="sold_out"?"Sube el cupo"
       :item.quantityControl==="inventory"&&item.source==="inventory"&&item.status==="sold_out"?"Registra una entrada":"";
      const quotaError=portionChanged&&!portionValid?`Ingresa un entero de al menos ${portionMinimum}. Ya hay ${item.soldQuantity} vendidas.`:undefined;
      return <article className={`availability-card status-${item.status}`} aria-label={item.name} key={item.productId}>
       <div className="availability-product">
        <span className="availability-image">{item.imageUrl?<img src={item.imageUrl} alt="" loading="lazy"/>:<Icon name={item.quantityControl==="inventory"?"stock":"dish"} size={24}/>}</span>
        <b>{item.name}</b>
       </div>
       <div className="availability-cell availability-status-cell" data-label="Disponibilidad"><Status tone={tones[item.status]}>{labels[item.status]}</Status>{notice&&<p className="availability-notice"><Icon name="info" size={14}/>{notice}</p>}{statusCaption&&<small>{statusCaption}</small>}</div>
       {item.quantityControl!=="none"&&<div className="availability-cell availability-quota-cell" data-label="Cupo de hoy">
        {item.quantityControl==="portions"?<><div className="availability-quota-editor"><Input aria-label={`Cupo de hoy para ${item.name}. Mínimo ${portionMinimum}`} aria-invalid={!!quotaError} aria-describedby={quotaError?`quota-error-${item.productId}`:undefined} type="number" min={portionMinimum} step="1" inputMode="numeric" disabled={!canManage||derived||update.isPending} value={portionValue} onChange={event=>setPortions(value=>({...value,[item.productId]:event.target.value}))}/>{canManage&&portionChanged&&<Button className="availability-quota-save" icon="save" aria-label={`Guardar cupo de ${item.name}`} disabled={derived||update.isPending||!portionValid} onClick={()=>saveQuota(item)}>Guardar</Button>}</div>{quotaError&&<p className="field-error" id={`quota-error-${item.productId}`} role="alert">{quotaError}</p>}</>:<span className="availability-inventory-source"><Icon name="stock" size={18}/><span>Desde Inventario<small>Se actualiza con las entradas y ventas.</small></span></span>}
       </div>}
       <div className={`availability-cell availability-balance${item.quantityControl==="none"?" availability-unlimited":""}`}>
        {item.quantityControl==="none"?<><Icon name={derived?"info":"availability"} size={20}/><span>{derived?"Disponibilidad automática":"Sin límite de cantidad"}<small>{derived?"Según la ficha y sus componentes":"Control manual de disponibilidad"}</small></span></>:<><div><small>Vendidas</small><span>{formatRegionalNumber(item.soldQuantity,location?.country)}</span></div><div><small>Restantes</small><strong>{remainingAmount??"Sin dato"}{remainingAmount!==null&&item.quantityControl==="inventory"&&<small>{item.inventoryUnit??"und"}</small>}</strong></div></>}
       </div>
       <div className="availability-actions">
        {canManage&&!derived&&(manuallySoldOut||!quantityExhausted)&&<Button kind="secondary" className={manuallySoldOut?"availability-available-action":"availability-soldout-action"} icon={manuallySoldOut?"refresh":"circleMinus"} aria-label={manuallySoldOut?`Marcar ${item.name} como disponible`:`Marcar ${item.name} como agotado`} disabled={update.isPending} onClick={()=>setManualStatus(item,manuallySoldOut?"available":"sold_out")} loading={pending}>{pending?"Guardando…":manuallySoldOut?"Reactivar":"Agotar hoy"}</Button>}
        <Button kind="secondary" className="availability-history-action" icon="clock" aria-label={`Ver historial de ${item.name}`} data-tooltip="Ver historial" disabled={update.isPending} onClick={()=>setHistory(item)}>Historial</Button>
       </div>
      </article>;
     })}
    </div>
   </div>}
   {!sessionLoading&&canRead&&!query.isLoading&&!query.isError&&<Pagination page={page} size={size} total={query.data?.total??items.length} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
  </section>
  {change&&<AvailabilityReasonDialog change={change} options={(change.kind==="quota"?query.data?.quotaReasons:query.data?.soldOutReasons)??[]} busy={update.isPending} close={()=>setChange(null)} save={async value=>{if(canManage&&!update.isPending){try{await update.mutateAsync({...change,...value});}catch{/* onError comunica la respuesta del API. */}}}}/>}
  {history&&<AvailabilityHistoryDialog item={history} close={()=>setHistory(null)}/>}
 </>;
}

function AvailabilitySkeleton(){
 return <>
  <div className="availability-list-head availability-skeleton-head" aria-hidden="true">{Array.from({length:5},(_,index)=><span key={index}><i/></span>)}</div>
  <div className="availability-grid" aria-label="Cargando disponibilidad">
   {Array.from({length:6},(_,index)=><article className="availability-card availability-skeleton" key={index}>
    <div className="availability-product"><i className="availability-skeleton-image"/><span/></div>
    <div className="availability-cell availability-status-cell"><i className="availability-skeleton-status"/></div>
    <div className="availability-cell availability-quota-cell"><i className="availability-skeleton-control"/></div>
    <div className="availability-cell availability-balance"><i className="availability-skeleton-text"/><i className="availability-skeleton-text"/></div>
    <div className="availability-actions"><i className="availability-skeleton-control"/><i className="availability-skeleton-control"/></div>
   </article>)}
  </div>
 </>;
}
