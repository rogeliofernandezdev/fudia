"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,IconName,Input,PageHeader,Pagination,Select,Status} from "@/design-system";
import {apiFetch} from "@/shared/api/client";
import {ComandaCatalog} from "@/components/comanda-catalog";
import {useFeedback} from "@/providers/feedback-provider";
import {useSettings} from "@/providers/settings-context";
import {useSession} from "@/providers/session-context";

type Option={value:string;label:string};
type OrderItem={id:string;productId:string;name:string;qty:string;unitPrice:string;note:string};
type Order={id:string;code:string;channel:string;status:string;customerId:string;customerName:string;customerPhone:string;address:string;reference:string;tableId:string;tableName:string;notes:string;subtotal:string;deliveryFee:string;total:string;createdAt:string;updatedAt:string;items?:OrderItem[]};
type Response={items:Order[];total:number;channelCounts:Record<string,number>;channelOptions:Option[];statusOptions:Option[]};
type Product={id:string;name:string;price:string;categoryId:string|null;imageUrl:string|null};
type LineDraft={productId:string;name:string;qty:number;unitPrice:number;note:string};
type Draft={channel:string;customerName:string;customerPhone:string;address:string;reference:string;tableId:string;notes:string;deliveryFee:string;lines:LineDraft[]};

const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"store",recojo:"box",delivery:"truck",whatsapp:"share"};
const statusMeta:Record<string,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={nuevo:{label:"Nuevo",tone:"blue"},confirmado:{label:"Confirmado",tone:"blue"},preparando:{label:"Preparando",tone:"orange"},listo:{label:"Listo",tone:"green"},en_camino:{label:"En camino",tone:"orange"},entregado:{label:"Entregado",tone:"gray"},cancelado:{label:"Cancelado",tone:"gray"}};
const emptyDraft:Draft={channel:"mostrador",customerName:"",customerPhone:"",address:"",reference:"",tableId:"",notes:"",deliveryFee:"0",lines:[]};

function nextAction(o:Order):{status:string;label:string}|null{
 if(o.status==="nuevo")return{status:"confirmado",label:"Confirmar"};
 if(o.status==="confirmado")return{status:"preparando",label:"Iniciar"};
 if(o.status==="preparando")return{status:"listo",label:"Marcar listo"};
 if(o.status==="listo")return o.channel==="delivery"?{status:"en_camino",label:"En camino"}:{status:"entregado",label:"Entregar"};
 if(o.status==="en_camino")return{status:"entregado",label:"Entregar"};
 return null;
}
const cancellable=(o:Order)=>!["entregado","cancelado"].includes(o.status);
function parseIsoDate(iso:string):Date|null{
  if(!iso)return null;
  const normalized=iso.replace(/([+-]\d{2})$/,"$1:00");
  const d=new Date(normalized);
  if(!isNaN(d.getTime()))return d;
  const d2=new Date(iso);
  return isNaN(d2.getTime())?null:d2;
}
function timeAgo(iso:string){
  const d=parseIsoDate(iso);if(!d)return"—";
  const m=Math.floor((Date.now()-d.getTime())/60000);
  if(m<1)return"Justo ahora";if(m<60)return`Hace ${m} min`;
  const h=Math.floor(m/60);if(h<24)return`Hace ${h} h`;
  return new Intl.DateTimeFormat("es-PE",{dateStyle:"medium",timeStyle:"short"}).format(d);
}
const money=(v:string|number)=>Number(v).toFixed(2);

export function OrdersManager(){
 const qc=useQueryClient();const{notify}=useFeedback();const settings=useSettings();const{can}=useSession();const canManage=can("orders.manage");
 const[q,setQ]=useState("");const[channel,setChannel]=useState("");const[status,setStatus]=useState("abiertos");const[page,setPage]=useState(1);const[size,setSize]=useState(12);
 const[detailId,setDetailId]=useState<string|null>(null);const[draft,setDraft]=useState<Draft|null>(null);const[cancelTarget,setCancelTarget]=useState<Order|null>(null);
 const list=useQuery({queryKey:["orders",q,channel,status,page,size],queryFn:()=>apiFetch<Response>(`orders?q=${encodeURIComponent(q)}&channel=${channel}&status=${status}&page=${page}&pageSize=${size}`)});
 const detail=useQuery({queryKey:["order",detailId],queryFn:()=>apiFetch<Order>(`orders/${detailId}`),enabled:Boolean(detailId)});
 const invalidate=()=>{void qc.invalidateQueries({queryKey:["orders"]});void qc.invalidateQueries({queryKey:["order",detailId]})};
 const newDraft=()=>({...emptyDraft,channel:"mostrador",tableId:"",lines:[]});
 const advance=useMutation({mutationFn:(v:{id:string;status:string})=>apiFetch<Order>(`orders/${v.id}/status`,{method:"PATCH",body:JSON.stringify({status:v.status})}),onSuccess:()=>{invalidate();notify({tone:"success",title:"Pedido actualizado",message:"El estado del pedido fue actualizado."})},onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message})});
 const create=useMutation({mutationFn:(v:Draft)=>apiFetch<Order>("orders",{method:"POST",body:JSON.stringify({channel:v.channel,customerName:v.customerName,customerPhone:v.customerPhone,address:v.address,reference:v.reference,tableId:v.tableId,notes:v.notes,deliveryFee:Number(v.deliveryFee)||0,items:v.lines.map(l=>({productId:l.productId,name:l.name,qty:l.qty,unitPrice:l.unitPrice,note:l.note}))})}),onSuccess:o=>{setDraft(null);invalidate();setDetailId(o.id);notify({tone:"success",title:"Pedido registrado",message:`El pedido ${o.code} quedó en estado Nuevo.`})},onError:e=>notify({tone:"danger",title:"No se pudo registrar",message:e.message})});
 const cancel=useMutation({mutationFn:(o:Order)=>apiFetch<Order>(`orders/${o.id}/status`,{method:"PATCH",body:JSON.stringify({status:"cancelado"})}),onSuccess:()=>{setCancelTarget(null);invalidate();notify({tone:"success",title:"Pedido cancelado",message:"El pedido quedó marcado como cancelado."})},onError:e=>notify({tone:"danger",title:"No se pudo cancelar",message:e.message})});
 const items=list.data?.items??[];const counts=list.data?.channelCounts??{};const channelOptions=list.data?.channelOptions??[];
 const channelLabel=(v:string)=>channelOptions.find(o=>o.value===v)?.label??v;
 const openTotal=Object.values(counts).reduce((a,b)=>a+b,0);
 return <><PageHeader eyebrow="OPERACIÓN" title="Pedidos" description="Salón, mostrador, recojo, delivery y WhatsApp en una sola bandeja." action={canManage?<Button icon="plus" onClick={()=>setDraft(newDraft())}>Nuevo pedido</Button>:undefined}/>
 <section className="panel management orders-panel">
  <div className="orders-channels">
   <button className={"orders-ch"+(channel===""?" active":"")} onClick={()=>{setChannel("");setPage(1)}}><Icon name="receipt" size={17}/><span>Todos</span><b>{openTotal}</b></button>
   {channelOptions.map(o=><button key={o.value} className={"orders-ch"+(channel===o.value?" active":"")} onClick={()=>{setChannel(o.value);setPage(1)}}><Icon name={channelIcons[o.value]??"receipt"} size={17}/><span>{o.label}</span><b>{counts[o.value]??0}</b></button>)}
  </div>
  <div className="toolbar"><label><Icon name="search" size={18}/><input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar por código, cliente o teléfono..."/></label><select aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option><option value="abiertos">Abiertos</option>{(list.data?.statusOptions??[]).map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</select></div>
  {list.isLoading?<Loading/>:list.isError?<State icon="alert" title="No pudimos cargar los pedidos" text={list.error.message} action={()=>list.refetch()}/>:!items.length?<State icon="receipt" title="Sin pedidos" text={canManage?"Registra el primer pedido o ajusta los filtros.":"No hay pedidos que coincidan con los filtros."} action={canManage?()=>setDraft(newDraft()):undefined}/>:<div className="orders-list">{items.map(o=>{const meta=statusMeta[o.status]??{label:o.status,tone:"gray" as const};const action=nextAction(o);const done=o.status==="entregado"||o.status==="cancelado";return <article className={`order-card channel-${o.channel}${done?" done":""}`} key={o.id} onClick={()=>setDetailId(o.id)}>
   <div className="order-card-left">
    <span className={"order-channel-icon oc-"+o.channel}><Icon name={channelIcons[o.channel]??"receipt"} size={18}/></span>
    <div className="order-card-info">
     <div className="order-card-top">
      <strong>{o.customerName||"Sin nombre"}</strong>
      <span className="order-id">{o.code}</span>
      <span className="order-channel-tag"><Icon name={channelIcons[o.channel]??"receipt"} size={12}/>{channelLabel(o.channel)}</span>
      {o.tableName&&<span className="order-table-tag"><Icon name="utensils" size={12}/>{o.tableName}</span>}
     </div>
     <div className="order-card-meta">
      <Status tone={meta.tone}>{meta.label}</Status>
      <span className="order-time-tag"><Icon name="clock" size={12}/>{timeAgo(o.createdAt)}</span>
      {o.address&&<span className="order-address-preview" title={o.address}><Icon name="truck" size={12}/>{o.address}</span>}
      {o.notes&&<span className="order-notes-preview" title={o.notes}><Icon name="edit" size={12}/>{o.notes}</span>}
     </div>
    </div>
   </div>
   <div className="order-card-right">
    <div className="order-total-block">
     <small>TOTAL</small>
     <b className="order-total">{settings.currencySymbol} {money(o.total)}</b>
    </div>
    {canManage&&action&&<Button className="order-advance" onClick={e=>{e.stopPropagation();advance.mutate({id:o.id,status:action.status})}} disabled={advance.isPending}>{action.label}<Icon name="chevron" size={14}/></Button>}
    {o.status==="entregado"&&<span className="order-done"><Icon name="check" size={14}/>Entregado</span>}
   </div>
  </article>})}</div>}
  <Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>
 </section>
 {draft&&<ComandaView initial={draft} channels={channelOptions} busy={create.isPending} currencySymbol={settings.currencySymbol} close={()=>setDraft(null)} save={v=>create.mutate(v)} notify={notify}/>}
 {detailId&&<OrderDetail loading={detail.isLoading} order={detail.data} error={detail.error?.message} channels={channelOptions} currencySymbol={settings.currencySymbol} canManage={canManage} busy={advance.isPending} close={()=>setDetailId(null)} advance={st=>advance.mutate({id:detailId,status:st})} cancel={o=>setCancelTarget(o)}/>}
 <ConfirmDialog open={Boolean(cancelTarget)} title="Cancelar pedido" description={`El pedido ${cancelTarget?.code??""} quedará cancelado y no podrá reactivarse.`} tone="danger" confirmLabel="Cancelar pedido" pending={cancel.isPending} onCancel={()=>setCancelTarget(null)} onConfirm={()=>cancelTarget&&cancel.mutate(cancelTarget)}/></>;
}

function ComandaView({initial,channels,busy,currencySymbol,close,save,notify}:{initial:Draft;channels:Option[];busy:boolean;currencySymbol:string;close:()=>void;save:(v:Draft)=>void;notify:(n:{tone:"danger"|"success";title:string;message:string})=>void}){
 const[v,setV]=useState(initial);const[ticketOpen,setTicketOpen]=useState(false);
 const floor=useQuery({queryKey:["orders-floor"],queryFn:()=>apiFetch<{items:{id:string;name:string;zone:string;order:{id:string}|null}[]}>("orders/floor"),enabled:v.channel==="salon",staleTime:30000});
 const freeTables=(floor.data?.items??[]).filter(t=>!t.order||t.id===v.tableId);
 const tableName=(floor.data?.items??[]).find(t=>t.id===v.tableId)?.name??"";
 const qtyByProduct=Object.fromEntries(v.lines.map(l=>[l.productId,l.qty]));
 const patchLine=(i:number,p:Partial<LineDraft>)=>setV({...v,lines:v.lines.map((l,n)=>n===i?{...l,...p}:l)});
 const tap=(p:Product)=>setV(prev=>{const i=prev.lines.findIndex(l=>l.productId===p.id);if(i>=0)return{...prev,lines:prev.lines.map((l,n)=>n===i?{...l,qty:l.qty+1}:l)};return{...prev,lines:[...prev.lines,{productId:p.id,name:p.name,qty:1,unitPrice:Number(p.price)||0,note:""}]}});
 const untap=(p:Product)=>setV(prev=>({...prev,lines:prev.lines.map(l=>l.productId===p.id?{...l,qty:l.qty-1}:l).filter(l=>l.qty>0)}));
 const step=(i:number,d:number)=>setV(prev=>({...prev,lines:prev.lines.map((l,n)=>n===i?{...l,qty:l.qty+d}:l).filter(l=>l.qty>0)}));
 const subtotal=v.lines.reduce((a,l)=>a+l.qty*l.unitPrice,0);const fee=Number(v.deliveryFee)||0;const count=v.lines.reduce((a,l)=>a+l.qty,0);
 const needsAddress=v.channel==="delivery";const needsTable=v.channel==="salon";const showContact=v.channel!=="salon"&&v.channel!=="mostrador";
 const channelLabel=(c:string)=>channels.find(o=>o.value===c)?.label??c;
 const submit=()=>{if(!v.lines.length||v.lines.some(l=>!l.name)){notify({tone:"danger",title:"Pedido incompleto",message:"Toca los productos para agregarlos a la comanda."});return};if(needsAddress&&!v.address.trim()){notify({tone:"danger",title:"Falta dirección",message:"El pedido de delivery necesita una dirección de entrega."});return};if(needsTable&&!v.tableId){notify({tone:"danger",title:"Falta mesa",message:"Selecciona la mesa del pedido."});return}save(v)};
 return <div className="comanda" role="dialog" aria-modal="true" aria-labelledby="comanda-title">
  <header className="comanda-head">
   <button className="comanda-back" aria-label="Volver" onClick={close}><Icon name="chevronLeft" size={20}/></button>
   <div className="comanda-title">
    <small>Nueva comanda</small>
    <h2 id="comanda-title">{needsTable?(tableName?`Comanda para ${tableName}`:"Elige la mesa"):"Nueva comanda"}</h2>
   </div>
   <span className="comanda-channel-tag"><i/>{channelLabel(v.channel)}</span>
   <button className="comanda-close" aria-label="Cerrar" onClick={close}><Icon name="close" size={18}/></button>
  </header>
  <div className="comanda-body">
   <ComandaCatalog qtyByProduct={qtyByProduct} onPick={tap} onRemove={untap} currencySymbol={currencySymbol}/>
   <aside className={"comanda-ticket"+(ticketOpen?" open":"")}>
    <i className="comanda-ticket-grip" aria-hidden="true"/>
    <div className="comanda-receipt-head">
     <div className="comanda-ticket-headrow">
      <small className="comanda-eyebrow">Comanda</small>
      <button type="button" className="comanda-ticket-close" onClick={()=>setTicketOpen(false)} aria-label="Volver a la carta"><Icon name="close" size={16}/></button>
     </div>
     <div className="comanda-channels">
      {channels.map(o=><button key={o.value} type="button" className={"comanda-channel-btn"+(v.channel===o.value?" active":"")+" ch-"+o.value} onClick={()=>setV({...v,channel:o.value,tableId:o.value==="salon"?v.tableId:"",address:"",reference:"",deliveryFee:o.value==="delivery"?v.deliveryFee:"0"})}><Icon name={channelIcons[o.value]??"receipt"} size={14}/><span>{o.label}</span></button>)}
     </div>
     {needsTable&&<div className="comanda-table-picker"><label>Mesa de atención</label><Select value={v.tableId} onChange={e=>setV({...v,tableId:e.target.value})} aria-label="Mesa"><option value="">Selecciona una mesa libre</option>{freeTables.map(t=><option key={t.id} value={t.id}>{t.zone?`${t.zone} · `:""}{t.name}</option>)}</Select></div>}
     {showContact&&<div className="comanda-fields">
      <Input value={v.customerName} onChange={e=>setV({...v,customerName:e.target.value})} placeholder="Nombre del cliente" aria-label="Cliente"/>
      <Input value={v.customerPhone} onChange={e=>setV({...v,customerPhone:e.target.value})} placeholder="Teléfono de contacto" aria-label="Teléfono"/>
     </div>}
     {needsAddress&&<div className="comanda-fields">
      <Input value={v.address} onChange={e=>setV({...v,address:e.target.value})} placeholder="Dirección de entrega *" aria-label="Dirección"/>
      <Input value={v.reference} onChange={e=>setV({...v,reference:e.target.value})} placeholder="Referencia de llegada" aria-label="Referencia"/>
      <label className="comanda-fee"><span>Costo delivery ({currencySymbol})</span><Input type="number" min="0" step="0.5" value={v.deliveryFee} onChange={e=>setV({...v,deliveryFee:e.target.value})} aria-label="Costo de delivery"/></label>
     </div>}
    </div>
    <div className="comanda-receipt-lines">
     {!v.lines.length&&<div className="comanda-empty"><span className="comanda-empty-icon"><Icon name="receipt" size={28}/></span><b>Comanda sin ítems</b><p>Toca los platos del catálogo para añadirlos al pedido.</p></div>}
     {v.lines.map((l,i)=>
      <div className="comanda-receipt-line" key={l.productId||i}>
       <div className="comanda-receipt-row">
        <b>{l.name}</b>
        <span className="comanda-receipt-leader" aria-hidden="true"/>
        <em>{currencySymbol} {money(l.qty*l.unitPrice)}</em>
       </div>
       <div className="comanda-receipt-controls">
        <div className="comanda-receipt-stepper">
         <button type="button" onClick={()=>step(i,-1)} aria-label="Quitar uno"><Icon name="minus" size={12}/></button>
         <b>{l.qty}</b>
         <button type="button" onClick={()=>step(i,1)} aria-label="Agregar uno"><Icon name="plus" size={12}/></button>
        </div>
        <input value={l.note} onChange={e=>patchLine(i,{note:e.target.value})} placeholder="Nota de preparación (ej. sin cebolla)" aria-label="Nota"/>
       </div>
      </div>)}
     {v.lines.length>0&&<div className="comanda-notes-wrap"><label>Instrucciones generales del pedido</label><input className="comanda-notes" value={v.notes} onChange={e=>setV({...v,notes:e.target.value})} placeholder="Indicaciones para cocina o motorizado..." aria-label="Notas del pedido"/></div>}
    </div>
    <footer className="comanda-receipt-foot">
     <div className="comanda-receipt-total"><span>Subtotal</span><span className="comanda-receipt-leader" aria-hidden="true"/><b>{currencySymbol} {money(subtotal)}</b></div>
     {fee>0&&<div className="comanda-receipt-total"><span>Envío delivery</span><span className="comanda-receipt-leader" aria-hidden="true"/><b>{currencySymbol} {money(fee)}</b></div>}
     <div className="comanda-receipt-total grand"><span>Total a pagar</span><span className="comanda-receipt-leader" aria-hidden="true"/><b>{currencySymbol} {money(subtotal+fee)}</b></div>
    </footer>
    <i className="comanda-receipt-zigzag" aria-hidden="true"/>
   </aside>
  </div>
  <footer className="comanda-foot">
   <div className="comanda-foot-info"><Icon name="receipt" size={14}/><span><b>{count}</b> ítem{count===1?"":"s"} en la comanda</span></div>
   <button type="button" className="comanda-foot-finalize" onClick={submit} disabled={busy||!v.lines.length}>{busy?"Registrando…":<>FINALIZAR <b>{currencySymbol} {money(subtotal+fee)}</b></>}</button>
  </footer>
  <div className={"comanda-bar"+(v.lines.length?" ready":"")}>
   <button type="button" className="comanda-bar-info" onClick={()=>setTicketOpen(!ticketOpen)} aria-expanded={ticketOpen}><b>{count}</b> plato{count===1?"":"s"} · {currencySymbol} {money(subtotal+fee)}<Icon name="chevron" size={15}/></button>
   {v.lines.length>0&&<button type="button" className="comanda-bar-finish" onClick={submit} disabled={busy}>{busy?"…":"FINALIZAR"}</button>}
  </div>
 </div>;
}

function OrderDetail({loading,order,error,channels,currencySymbol,canManage,busy,close,advance,cancel}:{loading:boolean;order?:Order;error?:string;channels:Option[];currencySymbol:string;canManage:boolean;busy:boolean;close:()=>void;advance:(st:string)=>void;cancel:(o:Order)=>void}){
 const channelLabel=(v:string)=>channels.find(o=>o.value===v)?.label??v;
 const meta=order?statusMeta[order.status]??{label:order.status,tone:"gray" as const}:null;
 const action=order?nextAction(order):null;
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal order-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="order-detail-title"><div className="modal-accent"/>
  <header className="order-detail-head">
   <span className="modal-title-icon"><Icon name="receipt"/></span>
   <div>
    <div className="order-detail-head-top">
     <h2 id="order-detail-title">{order?.code??"Pedido"}</h2>
     {meta&&<Status tone={meta.tone}>{meta.label}</Status>}
    </div>
    <small>DETALLE DE LA COMANDA</small>
   </div>
   <button aria-label="Cerrar" onClick={close} className="order-detail-close"><Icon name="close"/></button>
  </header>
  {loading?<Loading/>:error?<State icon="alert" title="No pudimos cargar el detalle" text={error}/>:order&&meta&&<>
   <div className="order-detail-body">
    <section className="order-detail-summary">
     <div className="summary-card">
      <small>CANAL</small>
      <b><Icon name={channelIcons[order.channel]??"receipt"} size={13}/>{channelLabel(order.channel)}</b>
     </div>
     <div className="summary-card">
      <small>{order.tableName?"MESA":"CLIENTE"}</small>
      <b>{order.tableName||order.customerName||"—"}</b>
     </div>
     <div className="summary-card">
      <small>REGISTRADO</small>
      <b>{timeAgo(order.createdAt)}</b>
     </div>
     <div className="summary-card">
      <small>ESTADO</small>
      <Status tone={meta.tone}>{meta.label}</Status>
     </div>
    </section>
    {(order.customerPhone||order.address)&&<section className="customer-detail-section">
     <header><div><small>CONTACTO Y ENTREGA</small><h3>Datos del cliente</h3></div></header>
     <section className="customer-detail-fields">
      {order.customerPhone&&<div><small>TELÉFONO</small><b>{order.customerPhone}</b></div>}
      {order.address&&<div><small>DIRECCIÓN</small><b>{order.address}</b></div>}
      {order.reference&&<div><small>REFERENCIA</small><b>{order.reference}</b></div>}
     </section>
    </section>}
    <section className="customer-detail-section">
     <header><div><small>CONSUMO</small><h3>Productos del pedido</h3></div></header>
     <div className="order-detail-items">{(order.items??[]).map(it=><div className="order-detail-line" key={it.id}>
      <b className="line-qty">{Number(it.qty)}×</b>
      <div className="order-detail-line-info">
       <span>{it.name}</span>
       {it.note&&<em>Nota: {it.note}</em>}
      </div>
      <strong className="line-price">{currencySymbol} {money(Number(it.qty)*Number(it.unitPrice))}</strong>
     </div>)}</div>
     {order.notes&&<div className="customer-detail-notes"><b>Instrucciones de comanda</b><p>{order.notes}</p></div>}
    </section>
    <section className="order-detail-totals">
     <div className="order-detail-sub"><span>Subtotal</span><b>{currencySymbol} {money(order.subtotal)}</b></div>
     {Number(order.deliveryFee)>0&&<div className="order-detail-sub"><span>Delivery</span><b>{currencySymbol} {money(order.deliveryFee)}</b></div>}
     <div className="order-detail-grand"><span>Total</span><strong>{currencySymbol} {money(order.total)}</strong></div>
    </section>
   </div>
   {canManage&&<footer className="order-detail-actions">
    {action&&<Button className="order-detail-primary" disabled={busy} onClick={()=>advance(action.status)}>{action.label}<Icon name="chevron" size={15}/></Button>}
    {order.status==="entregado"&&<span className="order-detail-done"><Icon name="check" size={15}/>Pedido completado</span>}
    {cancellable(order)&&<Button kind="ghost" className="order-detail-cancel" disabled={busy} onClick={()=>cancel(order)}>Cancelar pedido</Button>}
   </footer>}
  </>}
 </section></div>;
}

function Loading(){return <div className="customers-loading" aria-label="Cargando"><i/><i/><i/><i/></div>}
function State({icon,title,text,action}:{icon:"alert"|"receipt";title:string;text:string;action?:()=>void}){return <div className="catalog-state"><span><Icon name={icon}/></span><b>{title}</b><p>{text}</p>{action&&<Button kind="ghost" onClick={action}>{icon==="receipt"?"Nuevo pedido":"Reintentar"}</Button>}</div>}
