"use client";
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
 const{location}=useSession();
 const[search,setSearch]=useState("");
 const[categoryId,setCategoryId]=useState("");
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(10);
 const[portions,setPortions]=useState<Record<string,string>>({});

 const query=useQuery({queryKey:["product-availability",search,categoryId,page,size],queryFn:()=>listAvailability({search,categoryId,page,pageSize:size})});
 const categories=useQuery({queryKey:["categories","availability-filter"],queryFn:listAvailabilityCategories});
 const update=useMutation({
  mutationFn:({item,status}:{item:AvailabilityItem;status:"available"|"sold_out"})=>{
   const portionQuantity=Number(portions[item.productId]??item.portionQuantity??0);
   if(item.quantityControl==="portions"&&status!=="sold_out"&&(!Number.isInteger(portionQuantity)||portionQuantity<1)){
    throw new Error("Ingresa una cantidad de porciones válida mayor que cero.");
   }
   return updateAvailability(item.productId,{
    status,
    portionQuantity:item.quantityControl==="portions"?portionQuantity:null,
    note:item.note,
   });
  },
  onSuccess:(_,variables)=>{
   setPortions(value=>{const next={...value};delete next[variables.item.productId];return next});
   void client.invalidateQueries({queryKey:["product-availability"]});
   void client.invalidateQueries({queryKey:["inventory"]});
   notify({tone:"success",title:"Disponibilidad actualizada",message:"El cambio ya aplica al local activo."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message}),
 });

 const items=query.data?.items??[];
 const businessDate=query.data?.businessDate
  ?formatRegionalCalendarDate(query.data.businessDate,location?.country,{weekday:"short",day:"2-digit",month:"short"})
  :"Cargando fecha…";

 return <>
  <PageHeader eyebrow="CARTA Y PRODUCCIÓN" title="Disponibilidad de la carta" description="Administra el cupo diario y consulta la disponibilidad real de cada producto en el local activo." action={<span className="availability-date"><Icon name="clock" size={16}/>{businessDate}</span>}/>
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

   {query.isLoading?<div className="availability-table-wrap hover-scroll" tabIndex={0}><AvailabilitySkeleton/></div>:query.isError?
    <div className="availability-state"><Icon name="alert" size={24}/><b>No pudimos cargar la carta</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="availability-state"><Icon name="box" size={24}/><b>Sin productos</b><p>No hay productos activos que coincidan con los filtros.</p></div>
   :<div className="availability-table-wrap hover-scroll" tabIndex={0}>
    <div className="availability-list-head" aria-hidden="true"><span>PRODUCTO</span><span>ESTADO</span><span>CONTROL DEL DÍA</span><span>ACCIONES</span></div>
    <div className="availability-grid">
     {items.map(item=>{
      const derived=item.source==="schedule"||item.source==="combo_components";
      const pending=update.isPending&&update.variables?.item.productId===item.productId;
      const portionValue=portions[item.productId]??String(item.portionQuantity??0);
      const portionChanged=item.quantityControl==="portions"&&portionValue!==String(item.portionQuantity??0);
      const manuallySoldOut=item.manualStatus==="sold_out";
      const quantityExhausted=(item.source==="portions"||item.source==="inventory")&&item.status==="sold_out";
      const inventoryAmount=item.remaining===null?"0":formatRegionalNumber(Number(item.remaining),location?.country,{maximumFractionDigits:3});
      return <article className={`availability-card status-${item.status}`} key={item.productId}>
       <header className="availability-product">
        <span className="availability-image">{item.imageUrl?<img src={item.imageUrl} alt=""/>:<Icon name="box" size={20}/>}</span>
        <div><b>{item.name}</b><small>{item.categoryName??"Sin categoría"}</small></div>
       </header>

       <div className="availability-status-cell">
        <Status tone={tones[item.status]}>{labels[item.status]}</Status>
        <small>{derived?"Estado automático":manuallySoldOut?"Cambio manual":"Estado operativo"}</small>
       </div>

       <div className="availability-control-cell">
        <div className={`availability-control-grid control-${item.quantityControl}`}>
         <span className="availability-control-kind"><small>CONTROL</small><b>{controlLabels[item.quantityControl]}</b></span>
         {item.quantityControl==="portions"&&<>
          <label><small>CUPO DE HOY</small><Input aria-label={`Cupo de hoy para ${item.name}`} type="number" min="1" step="1" inputMode="numeric" disabled={derived||pending} value={portionValue} onChange={event=>setPortions(value=>({...value,[item.productId]:event.target.value}))}/></label>
          <span><small>VENDIDAS</small><b>{item.soldQuantity}</b></span>
          <span><small>RESTANTES</small><b>{item.remaining??0}</b></span>
         </>}
         {item.quantityControl==="inventory"&&<>
          <span><small>EXISTENCIA</small><b>{inventoryAmount} {item.inventoryUnit??"und"}</b></span>
          <span><small>ORIGEN</small><b>Inventario</b></span>
         </>}
         {item.quantityControl==="none"&&<span><small>CANTIDAD</small><b>No se controla</b></span>}
        </div>

        {item.quantityControl==="inventory"&&!derived&&<p className="availability-derived inventory"><Icon name="stock" size={14}/>La existencia se actualiza desde Inventario y con las ventas.</p>}
        {derived&&<p className="availability-derived"><Icon name="alert" size={14}/>{item.source==="schedule"?"La ficha del producto lo mantiene fuera de horario.":"No hay suficientes opciones disponibles en una parte obligatoria."}</p>}
        {item.quantityControl==="portions"&&item.source==="portions"&&item.status==="sold_out"&&<p className="availability-derived"><Icon name="alert" size={14}/>Las porciones del día se agotaron. Aumenta el cupo y actualízalo para continuar vendiendo.</p>}
        {item.quantityControl==="inventory"&&item.source==="inventory"&&item.status==="sold_out"&&<p className="availability-derived"><Icon name="alert" size={14}/>No queda stock físico. Registra una nueva entrada en Inventario.</p>}
       </div>

       <footer className={item.quantityControl==="portions"?"availability-actions":"availability-actions single"}>
        {item.quantityControl==="portions"&&<Button kind={portionChanged?"primary":"secondary"} icon="check" disabled={derived||pending||!portionChanged} onClick={()=>update.mutate({item,status:item.manualStatus})}>{pending?"Actualizando…":"Actualizar cupo"}</Button>}
        <Button kind="secondary" className={manuallySoldOut?"availability-available-action":"availability-soldout-action"} icon="power" disabled={derived||pending||(!manuallySoldOut&&quantityExhausted)} onClick={()=>update.mutate({item,status:manuallySoldOut?"available":"sold_out"})}>{pending?"Guardando…":manuallySoldOut?"Marcar disponible":"Marcar agotado"}</Button>
       </footer>
      </article>;
     })}
    </div>
   </div>}
   {!query.isLoading&&!query.isError&&<Pagination page={page} size={size} total={query.data?.total??items.length} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
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
