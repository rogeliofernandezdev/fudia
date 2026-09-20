"use client";
import "./customers.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status,Textarea} from "@/design-system";
import type {Address,Customer,CustomerDraft,Option} from "../domain/types";
import {getCustomer,listCustomers,saveCustomer,setCustomerActive} from "../infrastructure/customers-api";
import {useFeedback} from "@/providers/feedback-provider";
import {useSettings} from "@/providers/settings-context";
import {useSession} from "@/providers/session-context";

const empty:CustomerDraft={customerType:"person",displayName:"",documentType:"DNI",documentNumber:"",phone:"",email:"",preferredChannel:"none",preferences:"",notes:"",marketingConsent:false,vipOverride:false,addresses:[]};
const segmentLabels:Record<string,string>={new:"Nuevo",recurrent:"Recurrente",frequent:"Frecuente",vip:"VIP"};

export function CustomersManager(){const qc=useQueryClient();const{notify}=useFeedback();const settings=useSettings();const{can}=useSession();const canManage=can("customers.manage");const[q,setQ]=useState("");const[status,setStatus]=useState("");const[segment,setSegment]=useState("");const[page,setPage]=useState(1);const[size,setSize]=useState(10);const[draft,setCustomerDraft]=useState<CustomerDraft|null>(null);const[draftLoading,setDraftLoading]=useState(false);const[detailId,setDetailId]=useState<string|null>(null);const[target,setTarget]=useState<Customer|null>(null);
 const list=useQuery({queryKey:["customers",q,status,segment,page,size],queryFn:()=>listCustomers({q,status,segment,page,pageSize:size})});const detail=useQuery({queryKey:["customer",detailId],queryFn:()=>getCustomer(detailId!),enabled:Boolean(detailId)});
 const save=useMutation({mutationFn:(v:CustomerDraft)=>saveCustomer(v),onSuccess:()=>{setCustomerDraft(null);void qc.invalidateQueries({queryKey:["customers"]});notify({tone:"success",title:"Cliente guardado",message:"Los datos del cliente quedaron actualizados."})},onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message})});
 const changeStatus=useMutation({mutationFn:(c:Customer)=>setCustomerActive(c.id,!c.active),onSuccess:()=>{setTarget(null);void qc.invalidateQueries({queryKey:["customers"]});notify({tone:"success",title:"Estado actualizado",message:"El cliente conserva su historial y preferencias."})},onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message})});
 async function edit(id:string){setDraftLoading(true);setCustomerDraft({...empty,id});try{const c=await getCustomer(id);setCustomerDraft({id:c.id,customerType:c.customerType,displayName:c.displayName,documentType:c.documentType,documentNumber:c.documentNumber,phone:c.phone,email:c.email,preferredChannel:c.preferredChannel,preferences:c.preferences,notes:c.notes,marketingConsent:c.marketingConsent,vipOverride:c.vipOverride,addresses:c.addresses??[]})}catch(e){setCustomerDraft(null);notify({tone:"danger",title:"No se pudo cargar",message:(e as Error).message})}finally{setDraftLoading(false)}}
 const items=list.data?.items??[];return <><PageHeader eyebrow="NEGOCIO" title="Clientes" description="Directorio maestro, preferencias e historial de relación con tus clientes." action={canManage?<Button icon="plus" onClick={()=>setCustomerDraft({...empty})}>Nuevo cliente</Button>:undefined}/><section className="panel management standardized-management customers-panel"><div className="toolbar"><label><Icon name="search" size={18}/><input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar por nombre, documento, teléfono o correo..."/></label><select aria-label="Filtrar por segmento" value={segment} onChange={e=>{setSegment(e.target.value);setPage(1)}}><option value="">Todos los segmentos</option>{(list.data?.segmentOptions??[]).map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</select><select aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option><option value="active">Activos</option><option value="inactive">Inactivos</option></select></div>
 {list.isLoading?<Loading/>:list.isError?<State icon="alert" title="No pudimos cargar los clientes" text={list.error.message} action={()=>list.refetch()}/>:!items.length?<State icon="users" title="Aún no hay clientes" text={canManage?"Registra el primer cliente o ajusta los filtros.":"No hay clientes que coincidan con los filtros."} action={canManage?()=>setCustomerDraft({...empty}):undefined}/>:<><div className="table-wrap hover-scroll"><table><thead><tr><th>CLIENTE</th><th>CONTACTO</th><th>SEGMENTO</th><th>ACTIVIDAD</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{items.map((c,i)=><tr className={i%2?"alternate":""} key={c.id}><td><span className={`row-icon r${i%3}`}><Icon name="users" size={18}/></span><b>{c.displayName}</b>{c.documentNumber?<small className="doc-beside">{c.documentType} {c.documentNumber}</small>:null}</td><td>{c.phone||c.email||"Sin contacto"}<small>{c.phone&&c.email?c.email:""}</small></td><td><Status tone={c.segment==="vip"?"blue":c.segment==="frequent"?"green":"gray"}>{segmentLabels[c.segment]}</Status></td><td><b>{c.visitCount} visitas</b><small>S/ {Number(c.totalSpent).toFixed(2)} acumulado</small></td><td><Status tone={c.active?"green":"gray"}>{c.active?"Activo":"Inactivo"}</Status></td><td><div className="table-actions"><RowActionButton action="view" onClick={()=>setDetailId(c.id)}/>{canManage&&<><RowActionButton action="edit" onClick={()=>void edit(c.id)}/><RowActionButton action={c.active?"deactivate":"activate"} onClick={()=>setTarget(c)}/></>}</div></td></tr>)}</tbody></table></div><div className="management-cards">{items.map(c=><article key={c.id}><header><span className="row-icon r0"><Icon name="users"/></span><div><b>{c.displayName}</b>{c.documentNumber?<small className="doc-beside">{c.documentType} {c.documentNumber}</small>:null}</div><Status tone={c.active?"green":"gray"}>{c.active?"Activo":"Inactivo"}</Status></header><dl><div><dt>Contacto</dt><dd>{c.phone||c.email||"Sin contacto"}</dd></div><div><dt>Segmento</dt><dd>{segmentLabels[c.segment]}</dd></div><div><dt>Actividad</dt><dd>{c.visitCount} visitas · S/ {Number(c.totalSpent).toFixed(2)}</dd></div></dl><footer><RowActionButton action="view" onClick={()=>setDetailId(c.id)}/>{canManage&&<><RowActionButton action="edit" onClick={()=>void edit(c.id)}/><RowActionButton action={c.active?"deactivate":"activate"} onClick={()=>setTarget(c)}/></>}</footer></article>)}</div></>}
 <Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/></section>{draft&&(draftLoading||list.isLoading?<RemoteModalSkeleton className="customer-modal" label="Cargando cliente" close={()=>{setCustomerDraft(null);setDraftLoading(false)}}/>:<CustomerDialog initial={draft} channels={list.data?.channelOptions??[]} busy={save.isPending} close={()=>setCustomerDraft(null)} save={v=>save.mutate(v)}/>)} {detailId&&<CustomerDetail loading={detail.isLoading} customer={detail.data} error={detail.error?.message} channels={list.data?.channelOptions??[]} currencySymbol={settings.currencySymbol} close={()=>setDetailId(null)}/>}<ConfirmDialog open={Boolean(target)} title={target?.active?"Desactivar cliente":"Activar cliente"} description={target?.active?`“${target?.displayName??""}” dejará de estar disponible; su historial se conservará.`:`“${target?.displayName??""}” volverá a estar disponible.`} tone={target?.active?"danger":"success"} confirmLabel={target?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setTarget(null)} onConfirm={()=>target&&changeStatus.mutate(target)}/></>}

function CustomerDialog({initial,channels,busy,close,save}:{initial:CustomerDraft;channels:Option[];busy:boolean;close:()=>void;save:(v:CustomerDraft)=>void}){
 const[v,setV]=useState(initial);
 const[showAddresses,setShowAddresses]=useState(initial.addresses.length>0);
 const[showExtras,setShowExtras]=useState(Boolean(initial.preferences||initial.notes||initial.marketingConsent||initial.vipOverride));
 const add=()=>{setShowAddresses(true);setV({...v,addresses:[...v.addresses,{label:"Principal",address:"",reference:"",district:"",city:"",countryCode:"PE",default:!v.addresses.length}]})};
 const patch=(i:number,p:Partial<Address>)=>setV({...v,addresses:v.addresses.map((x,n)=>n===i?{...x,...p}:x)});
 const removeAddress=(i:number)=>setV({...v,addresses:v.addresses.filter((_,n)=>n!==i)});
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal customer-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="users"/></span><div><small>{v.id?"EDITAR CLIENTE":"NUEVO CLIENTE"}</small><h2>Información del cliente</h2></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
  <form onSubmit={e=>{e.preventDefault();save(v)}}><div className="customer-form-body">
   <div className="form-grid customer-main-grid">
    <label>Tipo<Select value={v.customerType} onChange={e=>setV({...v,customerType:e.target.value as CustomerDraft["customerType"]})}><option value="person">Persona</option><option value="company">Empresa</option></Select></label>
    <label className="span-2">Nombre o razón social<Input autoFocus required value={v.displayName} onChange={e=>setV({...v,displayName:e.target.value})} placeholder="Ej. María Torres"/></label>
    <label>Tipo de documento<Select value={v.documentType} onChange={e=>setV({...v,documentType:e.target.value})}><option value="">Sin documento</option><option>DNI</option><option>CE</option><option>RUC</option><option>PASAPORTE</option></Select></label>
    <label>Número de documento<Input value={v.documentNumber} onChange={e=>setV({...v,documentNumber:e.target.value})} placeholder="Ej. 12345678"/></label>
    <label>Teléfono<Input value={v.phone} onChange={e=>setV({...v,phone:e.target.value})} placeholder="Ej. 987 654 321"/></label>
    <label>Correo electrónico<Input type="email" value={v.email} onChange={e=>setV({...v,email:e.target.value})} placeholder="cliente@correo.com"/></label>
   </div>
   <button type="button" className={showAddresses?"product-extras-toggle open":"product-extras-toggle"} onClick={()=>setShowAddresses(!showAddresses)} aria-expanded={showAddresses}><Icon name="chevron" size={16}/><b>Direcciones</b><small>{v.addresses.length?`${v.addresses.length} ${v.addresses.length===1?"registrada":"registradas"}`:"Opcional"}</small></button>
   {showAddresses&&<div className="customer-addresses">
    {v.addresses.map((a,i)=><div className="customer-address" key={a.id??i}>
     <div className="customer-address-head">
      <Input value={a.label} onChange={e=>patch(i,{label:e.target.value})} placeholder="Etiqueta (Casa, Trabajo...)"/>
      <label className="customer-default"><input type="radio" name="default-address" checked={a.default} onChange={()=>setV({...v,addresses:v.addresses.map((x,n)=>({...x,default:n===i}))})}/><span>Predeterminada</span></label>
      <button type="button" className="customer-address-remove" onClick={()=>removeAddress(i)} aria-label="Quitar dirección"><Icon name="close" size={14}/></button>
     </div>
     <div className="form-grid">
      <label className="span-2">Dirección<Input required value={a.address} onChange={e=>patch(i,{address:e.target.value})} placeholder="Calle, número, interior"/></label>
      <label>Distrito<Input value={a.district} onChange={e=>patch(i,{district:e.target.value})}/></label>
      <label>Ciudad<Input value={a.city} onChange={e=>patch(i,{city:e.target.value})}/></label>
      <label className="span-2">Referencia<Input value={a.reference} onChange={e=>patch(i,{reference:e.target.value})} placeholder="Punto de referencia para el repartidor"/></label>
     </div>
    </div>)}
    <Button icon="plus" onClick={add}>Agregar dirección</Button>
   </div>}
   <button type="button" className={showExtras?"product-extras-toggle open":"product-extras-toggle"} onClick={()=>setShowExtras(!showExtras)} aria-expanded={showExtras}><Icon name="chevron" size={16}/><b>Información adicional</b><small>Opcional</small></button>
   {showExtras&&<div className="form-grid customer-extras">
    <label>Canal preferido<Select value={v.preferredChannel} onChange={e=>setV({...v,preferredChannel:e.target.value})}><option value="none">Sin preferencia</option>{channels.map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</Select></label>
    <div/>
    <label className="span-2">Preferencias<Textarea rows={2} value={v.preferences} onChange={e=>setV({...v,preferences:e.target.value})} placeholder="Ej. Sin picante, mesa junto a la ventana"/></label>
    <label className="span-2">Notas internas<Textarea rows={2} value={v.notes} onChange={e=>setV({...v,notes:e.target.value})} placeholder="Notas visibles solo para el equipo"/></label>
    <label className="switch-row compact"><input type="checkbox" checked={v.marketingConsent} onChange={e=>setV({...v,marketingConsent:e.target.checked})}/><span/><b>Acepta comunicaciones</b></label>
    <label className="switch-row compact"><input type="checkbox" checked={v.vipOverride} onChange={e=>setV({...v,vipOverride:e.target.checked})}/><span/><b>Marcar como VIP</b></label>
   </div>}
  </div><footer><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button></footer></form></section></div>
}

function CustomerDetail({customer,loading,error,channels,currencySymbol,close}:{customer?:Customer;loading:boolean;error?:string;channels:Option[];currencySymbol:string;close:()=>void}){
 const channelLabel=(value:string)=>value==="none"?"Sin preferencia":channels.find(o=>o.value===value)?.label??value;
 const lastPurchase=customer?.lastPurchaseAt?new Intl.DateTimeFormat("es-PE",{dateStyle:"medium"}).format(new Date(customer.lastPurchaseAt)):"—";
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal customer-detail modal-panel-in" role="dialog" aria-modal="true" aria-busy={loading}>
  <div className="modal-accent"/>
  {loading?<CustomerDetailSkeleton close={close}/>:<>
   <header><span className="modal-title-icon"><Icon name="users"/></span><div><h2>{customer?.displayName??"Cliente"}</h2><small>DETALLE DEL CLIENTE</small></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
   {error?<State icon="alert" title="No pudimos cargar el detalle" text={error}/>:customer&&<div className="customer-detail-body">
    <section className="customer-detail-summary">
     <div><small>SEGMENTO</small><b>{segmentLabels[customer.segment]}</b></div>
     <div><small>VISITAS</small><b>{customer.visitCount}</b></div>
     <div><small>ACUMULADO</small><b>{currencySymbol} {Number(customer.totalSpent).toFixed(2)}</b></div>
     <div><small>ÚLTIMA COMPRA</small><b>{lastPurchase}</b></div>
     <Status tone={customer.active?"green":"gray"}>{customer.active?"Activo":"Inactivo"}</Status>
    </section>
    <section className="customer-detail-fields">
     <div><small>DOCUMENTO</small><b>{customer.documentNumber?`${customer.documentType} ${customer.documentNumber}`:"No registrado"}</b></div>
     <div><small>TELÉFONO</small><b>{customer.phone||"—"}</b></div>
     <div><small>CORREO</small><b>{customer.email||"—"}</b></div>
     <div><small>CANAL PREFERIDO</small><b>{channelLabel(customer.preferredChannel)}</b></div>
     <div><small>COMUNICACIONES</small><b>{customer.marketingConsent?"Autorizadas":"No autorizadas"}</b></div>
     <div><small>CLIENTE VIP</small><b>{customer.vipOverride?"Sí":"No"}</b></div>
    </section>
    <section className="customer-detail-section">
     <header><div><small>ENTREGAS</small><h3>Direcciones de entrega registradas</h3></div></header>
     {customer.addresses?.length?customer.addresses.map(a=><div className="customer-detail-address" key={a.id}><div><b>{a.label}</b>{a.default&&<span>Predeterminada</span>}</div><p>{a.address}{a.district?`, ${a.district}`:""}{a.city?`, ${a.city}`:""}</p>{a.reference&&<small>{a.reference}</small>}</div>):<p className="customer-detail-empty">No registradas.</p>}
    </section>
    <section className="customer-detail-section">
     <header><div><small>RELACIÓN</small><h3>Preferencias y notas internas</h3></div></header>
     <div className="customer-detail-notes"><b>Preferencias</b><p>{customer.preferences||"Sin preferencias registradas."}</p><b>Notas internas</b><p>{customer.notes||"Sin notas."}</p></div>
    </section>
   </div>}
  </>}
 </section></div>
}

function CustomerDetailSkeleton({close}:{close:()=>void}){
 return <>
  <header className="customer-detail-skeleton-head" aria-hidden="true">
   <span className="customer-detail-skeleton-block customer-detail-skeleton-icon"/>
   <div className="customer-detail-skeleton-copy"><i/><b/></div>
   <button aria-label="Cerrar" onClick={close}><Icon name="close"/></button>
  </header>
  <div className="customer-detail-body customer-detail-skeleton-body" aria-label="Cargando detalle del cliente">
   <section className="customer-detail-summary customer-detail-skeleton-summary">
    {Array.from({length:4},(_,i)=><div key={i}><i/><b/></div>)}
    <span className="customer-detail-skeleton-block customer-detail-skeleton-status"/>
   </section>
   <section className="customer-detail-fields customer-detail-skeleton-fields">
    {Array.from({length:6},(_,i)=><div key={i}><i/><b/></div>)}
   </section>
   {Array.from({length:2},(_,i)=><section className="customer-detail-section customer-detail-skeleton-section" key={i}>
    <header><div><i/><b/></div></header>
    <div className="customer-detail-skeleton-lines"><i/><i/><i/></div>
   </section>)}
  </div>
 </>;
}

function Loading(){return <div className="customers-loading" aria-label="Cargando"><i/><i/><i/><i/></div>}function State({icon,title,text,action}:{icon:"alert"|"users";title:string;text:string;action?:()=>void}){return <div className="catalog-state"><span><Icon name={icon}/></span><b>{title}</b><p>{text}</p>{action&&<Button kind="ghost" onClick={action}>{icon==="users"?"Nuevo cliente":"Reintentar"}</Button>}</div>}
