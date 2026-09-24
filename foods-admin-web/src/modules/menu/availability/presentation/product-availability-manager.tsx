"use client";
/* eslint-disable @next/next/no-img-element -- Product images use tenant-configured URLs outside Next image domains. */
import "./product-availability.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,Select,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSession} from "@/providers/session-context";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {AvailabilityItem,AvailabilityStatus} from "../domain/types";
import {listAvailability,listAvailabilityCategories,updateAvailability} from "../infrastructure/availability-api";

const labels:Record<AvailabilityStatus,string>={available:"Disponible",low:"Pocas unidades",sold_out:"Agotado",unavailable:"Fuera de horario"};
const tones:Record<AvailabilityStatus,"green"|"orange"|"gray">={available:"green",low:"orange",sold_out:"gray",unavailable:"gray"};
const controlLabels:Record<AvailabilityItem["quantityControl"],string>={
  none:"Sin control",
  portions:"Porciones preparadas",
  inventory:"Inventario físico",
};

export function ProductAvailabilityManager(){
 const client=useQueryClient();
 const{notify}=useFeedback();
 const{location,can,isLoading:sessionLoading}=useSession();
 const[search,setSearch]=useState("");
 const[categoryId,setCategoryId]=useState("");
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(10);
 const[portions,setPortions]=useState<Record<string,string>>({});
 const canRead=can("menu.read");
 const canManage=can("menu.manage");

 const query=useQuery({
  queryKey:["product-availability",location?.id,search,categoryId,page,size],
  queryFn:()=>listAvailability({search,categoryId,page,pageSize:size}),
  enabled:!!location?.id&&canRead,
 });
 const categories=useQuery({queryKey:["categories","availability-filter"],queryFn:listAvailabilityCategories,enabled:canRead});
 const update=useMutation({
  mutationFn:({item,status,portionQuantity}:{item:AvailabilityItem;status:"available"|"sold_out";portionQuantity:number|null;kind:"quota"|"status"})=>
   updateAvailability(item.productId,{status,portionQuantity,note:item.note}),
  onSuccess:(_,variables)=>{
   if(variables.kind==="quota"){
    setPortions(value=>{const next={...value};delete next[variables.item.productId];return next});
   }
   void client.invalidateQueries({queryKey:["product-availability"]});
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
  if(!Number.isInteger(portionQuantity)||portionQuantity<minimum){
   notify({tone:"danger",title:"Cupo no válido",message:`El cupo de hoy debe ser un entero de al menos ${minimum}, porque ya hay ${item.soldQuantity} vendidas.`});
   return;
  }
  update.mutate({item,status:item.manualStatus,portionQuantity,kind:"quota"});
 }

 function setManualStatus(item:AvailabilityItem,status:"available"|"sold_out"){
  update.mutate({item,status,portionQuantity:null,kind:"status"});
 }

 const items=query.data?.items??[];
 const businessDate=query.data?.businessDate
  ?formatRegionalCalendarDate(query.data.businessDate,location?.country,{weekday:"short",day:"2-digit",month:"short"})
  :"Cargando fecha…";

 return <>
  <PageHeader eyebrow="CARTA Y PRODUCCIÓN" title="Disponibilidad" description="Administra el cupo diario y consulta la disponibilidad real de cada producto en el local activo." action={<span className="availability-date"><Icon name="clock" size={16}/>{businessDate}</span>}/>
  <section className="panel availability-panel">
   <header className="availability-toolbar">
    <div className="availability-filters">
     <label className="availability-search"><Icon name="search" size={18}/><Input value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar plato o producto..."/></label>
     <Select value={categoryId} onChange={event=>{setCategoryId(event.target.value);setPage(1)}} disabled={categories.isLoading} aria-label="Filtrar por categoría">
      <option value="">Todas las categorías</option>
      {categories.data?.items.map(category=><option value={category.id} key={category.id}>{category.name}</option>)}
     </Select>
    </div>
    <p className="availability-context"><Icon name="store" size={16}/>Las cantidades corresponden al local activo.</p>
   </header>

   {sessionLoading?<div className="availability-table-wrap hover-scroll" tabIndex={0}><AvailabilitySkeleton/></div>:!canRead?
    <div className="availability-state"><Icon name="alert" size={24}/><b>Sin permiso de lectura</b><p>Tu rol no permite consultar la disponibilidad.</p></div>
   :query.isLoading?<div className="availability-table-wrap hover-scroll" tabIndex={0}><AvailabilitySkeleton/></div>:query.isError?
    <div className="availability-state"><Icon name="alert" size={24}/><b>No pudimos cargar la carta</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="availability-state"><Icon name="availability" size={24}/><b>Sin productos</b><p>No hay productos activos que coincidan con los filtros.</p></div>
   :<div className="availability-table-wrap hover-scroll" tabIndex={0}>
    <div className="availability-list-head" aria-hidden="true"><span>PRODUCTO</span><span>ESTADO</span><span>CONTROL DEL DÍA</span><span>ACCIONES</span></div>
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
      const inventoryAmount=item.remaining===null?"0":formatRegionalNumber(Number(item.remaining),location?.country,{maximumFractionDigits:3});
      const statusCaption=derived?"Calculado automáticamente":manuallySoldOut?"Agotado manualmente":item.quantityControl==="portions"?"Según cupo del día":item.quantityControl==="inventory"?"Según inventario":"Control manual";
      return <article className={`availability-card status-${item.status}`} key={item.productId}>
       <header className="availability-product">
        <span className="availability-image">{item.imageUrl?<img src={item.imageUrl} alt=""/>:<Icon name="box" size={20}/>}</span>
        <div><b>{item.name}</b><small>{item.categoryName??"Sin categoría"}</small></div>
       </header>

       <div className="availability-status-cell">
        <Status tone={tones[item.status]}>{labels[item.status]}</Status>
        <small>{statusCaption}</small>
       </div>

       <div className="availability-control-cell">
        <div className={`availability-control-grid control-${item.quantityControl}`}>
         <span className="availability-control-kind"><small>CONTROL</small><b>{controlLabels[item.quantityControl]}</b></span>
         {item.quantityControl==="portions"&&<>
          <label><small>CUPO DE HOY</small><Input aria-label={`Cupo de hoy para ${item.name}. Mínimo ${portionMinimum}`} type="number" min={portionMinimum} step="1" inputMode="numeric" disabled={!canManage||derived||update.isPending} value={portionValue} onChange={event=>setPortions(value=>({...value,[item.productId]:event.target.value}))}/></label>
          <span><small>VENDIDAS</small><b>{item.soldQuantity}</b></span>
          <span><small>RESTANTES</small><b>{item.remaining??0}</b></span>
         </>}
         {item.quantityControl==="inventory"&&<>
          <span><small>EXISTENCIA</small><b>{inventoryAmount} {item.inventoryUnit??"und"}</b></span>
          <span><small>ORIGEN</small><b>Inventario</b></span>
         </>}
         {item.quantityControl==="none"&&<span><small>CANTIDAD</small><b>No se controla</b></span>}
        </div>
       </div>

       {item.quantityControl==="inventory"&&!derived&&<p className="availability-derived inventory"><Icon name="stock" size={14}/>La existencia se actualiza desde Inventario y con las ventas.</p>}
       {derived&&<p className="availability-derived"><Icon name="alert" size={14}/>{item.source==="schedule"?"La ficha del producto lo mantiene fuera de horario.":"No hay suficientes opciones disponibles en una parte obligatoria."}</p>}
       {item.quantityControl==="portions"&&item.source==="portions"&&item.status==="sold_out"&&<p className="availability-derived"><Icon name="alert" size={14}/>Cupo agotado. Aumenta el cupo de hoy y guarda el cambio para continuar vendiendo.</p>}
       {item.quantityControl==="inventory"&&item.source==="inventory"&&item.status==="sold_out"&&<p className="availability-derived"><Icon name="alert" size={14}/>No queda stock físico. Registra una nueva entrada en Inventario.</p>}

       <footer className={canManage&&item.quantityControl==="portions"?"availability-actions":"availability-actions single"}>
        {canManage?<>
         {item.quantityControl==="portions"&&<Button kind={portionChanged&&portionValid?"primary":"secondary"} icon="check" disabled={derived||update.isPending||!portionChanged||!portionValid} onClick={()=>saveQuota(item)} aria-label={`Guardar cupo de ${item.name}`}>{pending?"Guardando…":"Guardar"}</Button>}
         <Button kind="secondary" className={manuallySoldOut?"availability-available-action":"availability-soldout-action"} icon="power" aria-label={manuallySoldOut?`Marcar ${item.name} como disponible`:`Marcar ${item.name} como agotado`} disabled={derived||update.isPending||(!manuallySoldOut&&quantityExhausted)} onClick={()=>setManualStatus(item,manuallySoldOut?"available":"sold_out")}>{pending?"Guardando…":manuallySoldOut?"Reactivar":"Agotar hoy"}</Button>
        </>:<span className="availability-read-only">Solo lectura</span>}
       </footer>
      </article>;
     })}
    </div>
   </div>}
   {!sessionLoading&&canRead&&!query.isLoading&&!query.isError&&<Pagination page={page} size={size} total={query.data?.total??items.length} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
  </section>
 </>;
}

function AvailabilitySkeleton(){
 return <>
  <div className="availability-list-head availability-skeleton-head" aria-hidden="true"><span><i/></span><span><i/></span><span><i/></span><span><i/></span></div>
  <div className="availability-grid" aria-label="Cargando disponibilidad">
   {Array.from({length:6},(_,index)=><article className="availability-card availability-skeleton" key={index}>
    <header className="availability-product"><i className="availability-skeleton-image"/><div><span/><em/></div></header>
    <div className="availability-status-cell"><i className="availability-skeleton-status"/><em/></div>
    <div className="availability-control-cell"><div className="availability-skeleton-metrics"><i/><i/><i/><i/></div></div>
    <footer className="availability-actions"><i/><i/></footer>
   </article>)}
  </div>
 </>;
}
