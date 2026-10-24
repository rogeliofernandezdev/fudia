"use client";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import {LocationMap} from "@/design-system/location-map";
import "./customers.css";
import {useRef,useState} from "react";import {useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,FormField,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,SearchField,Select,Status,TableSkeleton,ModalHeader,ModalForm,ModalBody,ModalFooter,FormSection,DetailHero,DetailContact,DetailFacts} from "@/design-system";
import {customerResolver} from "../domain/customer-schema";import type {Address,Customer,CustomerDraft,Option} from "../domain/types";
import {getCustomer,listCustomers,saveCustomer,setCustomerActive} from "../infrastructure/customers-api";
import {useFeedback} from "@/providers/feedback-provider";
import {useMoney} from "@/providers";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {useSession} from "@/providers/session-context";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";

const empty:CustomerDraft={customerType:"person",displayName:"",documentType:"DNI",documentNumber:"",phone:"",email:"",preferredChannel:"none",preferences:"",notes:"",marketingConsent:false,vipOverride:false,addresses:[]};
const segmentLabels:Record<string,string>={new:"Nuevo",recurrent:"Recurrente",frequent:"Frecuente",vip:"VIP"};
const columns=["Cliente","Contacto","Segmento","Actividad","Estado","Acciones"];

export function CustomersManager(){const qc=useQueryClient();const{notify}=useFeedback();const money=useMoney();const{can}=useSession();const canManage=can("customers.manage");const[q,setQ]=useState("");const debouncedQ=useDebouncedValue(q);const[status,setStatus]=useState("");const[segment,setSegment]=useState("");const[page,setPage]=useState(1);const[size,setSize]=useState(10);const[draft,setCustomerDraft]=useState<CustomerDraft|null>(null);const[draftLoading,setDraftLoading]=useState(false);const cancelledDraft=useRef<string|null>(null);const[detailId,setDetailId]=useState<string|null>(null);const[target,setTarget]=useState<Customer|null>(null);
 const list=useQuery({queryKey:["customers",debouncedQ,status,segment,page,size],queryFn:()=>listCustomers({q:debouncedQ,status,segment,page,pageSize:size})});const detail=useQuery({queryKey:["customer",detailId],queryFn:()=>getCustomer(detailId!),enabled:Boolean(detailId)});
 const save=useMutation({mutationFn:(v:CustomerDraft)=>saveCustomer(v),onSuccess:()=>{setCustomerDraft(null);void qc.invalidateQueries({queryKey:["customers"]});notify({tone:"success",title:"Cliente guardado",message:"Los datos del cliente quedaron actualizados."})},onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message})});
 const changeStatus=useMutation({mutationFn:(c:Customer)=>setCustomerActive(c.id,!c.active),onSuccess:()=>{setTarget(null);void qc.invalidateQueries({queryKey:["customers"]});notify({tone:"success",title:"Estado actualizado",message:"El cliente conserva su historial y preferencias."})},onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message})});
 async function edit(id:string){cancelledDraft.current=null;setDraftLoading(true);setCustomerDraft({...empty,id});try{const c=await getCustomer(id);if(cancelledDraft.current===id){cancelledDraft.current=null;return}setCustomerDraft({id:c.id,customerType:c.customerType,displayName:c.displayName,documentType:c.documentType,documentNumber:c.documentNumber,phone:c.phone,email:c.email,preferredChannel:c.preferredChannel,preferences:c.preferences,notes:c.notes,marketingConsent:c.marketingConsent,vipOverride:c.vipOverride,addresses:c.addresses??[]})}catch(e){if(cancelledDraft.current!==id){setCustomerDraft(null);notify({tone:"danger",title:"No se pudo cargar",message:(e as Error).message})}}finally{setDraftLoading(false)}}
 const items=list.data?.items??[];const filtered=Boolean(debouncedQ||status||segment);return <><PageHeader title="Clientes" description="Directorio maestro, preferencias e historial de relación con tus clientes." action={canManage?<Button icon="plus" onClick={()=>setCustomerDraft({...empty})}>Nuevo cliente</Button>:undefined}/><section className="panel management standardized-management customers-panel"><div className="toolbar"><SearchField value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar por nombre, documento, teléfono o correo…" label="Buscar clientes"/><Select chip aria-label="Filtrar por segmento" value={segment} onChange={e=>{setSegment(e.target.value);setPage(1)}}><option value="">Todos los segmentos</option>{(list.data?.segmentOptions??[]).map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</Select><Select chip aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option><option value="active">Activos</option><option value="inactive">Inactivos</option></Select></div>
 {list.isLoading?<TableSkeleton columns={columns} label="Cargando clientes"/>:list.isError?<ErrorState title="No pudimos cargar los clientes" text={list.error.message} onRetry={()=>void list.refetch()} retrying={list.isFetching}/>:!items.length?(filtered?<EmptyState icon="search" title="Sin coincidencias" text="No hay clientes que coincidan con la búsqueda o los filtros."/>:<EmptyState icon="users" title="Aún no hay clientes" text={canManage?"Registra el primer cliente para guardar sus datos y preferencias.":"Los clientes registrados aparecerán aquí."} action={canManage?<Button icon="plus" onClick={()=>setCustomerDraft({...empty})}>Nuevo cliente</Button>:undefined}/>):<><DataTable label="Listado de clientes" containerClassName="hover-scroll"><thead><tr>{columns.map(c=><th key={c}>{c}</th>)}</tr></thead><tbody>{items.map((c,i)=><tr className={i%2?"alternate":""} key={c.id}><td><span className={`row-icon r${i%3}`}><Icon name="users" size={18}/></span><b>{c.displayName}</b>{c.documentNumber?<small className="doc-beside">{c.documentType} {c.documentNumber}</small>:null}</td><td>{c.phone||c.email||"Sin contacto"}<small>{c.phone&&c.email?c.email:""}</small></td><td><Status tone={c.segment==="vip"?"blue":c.segment==="frequent"?"green":"gray"}>{segmentLabels[c.segment]}</Status></td><td><b>{c.visitCount} visitas</b><small>{money(c.totalSpent)} acumulado</small></td><td><Status active={c.active}>{c.active?"Activo":"Inactivo"}</Status></td><td><div className="table-actions"><RowActionButton action="view" onClick={()=>setDetailId(c.id)}/>{canManage&&<><RowActionButton action="edit" onClick={()=>void edit(c.id)}/><RowActionButton action={c.active?"deactivate":"activate"} stateLabel={`Estado de ${c.displayName}`} onClick={()=>setTarget(c)}/></>}</div></td></tr>)}</tbody></DataTable><div className="management-cards">{items.map(c=><article key={c.id}><header><span className="row-icon r0"><Icon name="users"/></span><div><b>{c.displayName}</b>{c.documentNumber?<small className="doc-beside">{c.documentType} {c.documentNumber}</small>:null}</div><Status active={c.active}>{c.active?"Activo":"Inactivo"}</Status></header><dl><div><dt>Contacto</dt><dd>{c.phone||c.email||"Sin contacto"}</dd></div><div><dt>Segmento</dt><dd>{segmentLabels[c.segment]}</dd></div><div><dt>Actividad</dt><dd>{c.visitCount} visitas · {money(c.totalSpent)}</dd></div></dl><footer><RowActionButton action="view" onClick={()=>setDetailId(c.id)}/>{canManage&&<><RowActionButton action="edit" onClick={()=>void edit(c.id)}/><RowActionButton action={c.active?"deactivate":"activate"} stateLabel={`Estado de ${c.displayName}`} onClick={()=>setTarget(c)}/></>}</footer></article>)}</div></>}
 <Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/></section>{draft&&(draftLoading||list.isLoading?<RemoteModalSkeleton className="customer-modal" label="Cargando cliente" close={()=>{if(draft.id)cancelledDraft.current=draft.id;setCustomerDraft(null);setDraftLoading(false)}}/>:<CustomerDialog initial={draft} documentTypes={list.data?.documentTypeOptions??[]} busy={save.isPending} close={()=>setCustomerDraft(null)} save={v=>save.mutate(v)}/>)} {detailId&&<CustomerDetail loading={detail.isLoading} customer={detail.data} error={detail.error?.message} retry={()=>void detail.refetch()} retrying={detail.isFetching} close={()=>setDetailId(null)}/>}<ConfirmDialog open={Boolean(target)} title={target?.active?"Desactivar cliente":"Activar cliente"} subject={target?{label:"Cliente",name:target.displayName}:undefined} description={target?.active?"Dejará de estar disponible; su historial y preferencias se conservarán.":"Volverá a estar disponible para nuevos pedidos."} tone={target?.active?"danger":"success"} confirmLabel={target?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setTarget(null)} onConfirm={()=>target&&changeStatus.mutate(target)}/></>}

const blankAddress:Address={label:"Principal",address:"",reference:"",district:"",city:"",countryCode:"PE",default:true,latitude:null,longitude:null};
/** El formulario edita una sola dirección: la predeterminada va primero; las demás se conservan sin cambios al guardar. */
function withPrimaryAddress(addresses:Address[]):Address[]{
 const index=Math.max(0,addresses.findIndex(a=>a.default));
 const primary=addresses[index]??blankAddress;
 return [{...primary,default:true},...addresses.filter((_,n)=>n!==index).map(a=>({...a,default:false}))];
}

function CustomerDialog({initial,documentTypes,busy,close,save}:{initial:CustomerDraft;documentTypes:Option[];busy:boolean;close:()=>void;save:(v:CustomerDraft)=>void}){
 const{register,watch,setValue,handleSubmit,formState:{errors}}=useForm<CustomerDraft>({defaultValues:{...initial,addresses:withPrimaryAddress(initial.addresses)},resolver:customerResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const address=watch("addresses.0");
 const addressError=errors.addresses?.[0];
 const documentType=watch("documentType");
 // El documento define el tipo: RUC es empresa; DNI, CE o pasaporte es persona; sin documento conserva el tipo guardado.
 const isCompany=documentType?documentType==="RUC":initial.customerType==="company";
 const submit=handleSubmit(values=>{values.customerType=isCompany?"company":"person";save(values)});
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal customer-modal modal-panel-in" role="dialog" aria-modal="true">
  <ModalHeader title="Información del cliente" icon="users" subtitle={initial.id?"Editar cliente":"Nuevo cliente"} onClose={close} busy={busy} closeLabel="Cerrar"/>
  <ModalForm onSubmit={submit} noValidate><ModalBody>
   <div className="form-sections-flow"><FormSection icon="user" title="Datos del cliente" bare><div className="form-grid customer-main-grid">
    <FormField label="Documento"><Select {...register("documentType")}>{documentTypes.map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</Select></FormField>
    <FormField label="Número de documento" error={errors.documentNumber?.message}><Input icon="badge" autoFocus inputMode={documentType==="DNI"||documentType==="RUC"?"numeric":undefined} disabled={!documentType} {...register("documentNumber")} placeholder={documentType==="RUC"?"Ej. 20123456789":documentType?"Ej. 12345678":"No aplica"}/></FormField>
    <FormField className="customer-field-full" label={isCompany?"Razón social":"Nombre"} error={errors.displayName?.message}><Input maxLength={180} {...register("displayName")} placeholder={isCompany?"Ej. Inversiones Torres S.A.C.":"Ej. María Torres"}/></FormField>
    <FormField className="customer-field-contact" label="Teléfono" error={errors.phone?.message}><Input icon="phone" type="tel" inputMode="tel" {...register("phone")} placeholder="Ej. 987 654 321"/></FormField>
    <FormField className="customer-field-contact" label="Correo electrónico" optional error={errors.email?.message}><Input icon="mail" type="email" inputMode="email" {...register("email")} placeholder="cliente@correo.com"/></FormField>
   </div></FormSection>
   <FormSection icon="mapPin" tone="teal" title="Dirección de entrega" bare><div className="form-grid customer-address">
    <div className="span-2 customer-address-map">
     <LocationMap addressLabel={<span className="form-field-label">Dirección<em>Opcional</em></span>} mapLabel="Mapa de la dirección de entrega" latitude={address?.latitude??null} longitude={address?.longitude??null} address={address?.address??""} onAddressChange={value=>setValue("addresses.0.address",value,{shouldDirty:true,shouldValidate:Boolean(addressError?.address)})} onChange={point=>{setValue("addresses.0.latitude",point.lat,{shouldDirty:true});setValue("addresses.0.longitude",point.lng,{shouldDirty:true})}} onPlaceChange={place=>{setValue("addresses.0.district",place.district,{shouldDirty:true});setValue("addresses.0.city",place.city,{shouldDirty:true})}}/>
     {addressError?.address?.message&&<small className="field-error" role="alert">{addressError.address.message}</small>}
    </div>
    <FormField label="Distrito"><Input readOnly tabIndex={-1} {...register("addresses.0.district")} placeholder="Según la dirección"/></FormField>
    <FormField label="Ciudad"><Input readOnly tabIndex={-1} {...register("addresses.0.city")} placeholder="Según la dirección"/></FormField>
    <FormField className="span-2" label="Referencia" optional error={addressError?.reference?.message}><Input {...register("addresses.0.reference")} placeholder="Punto de referencia para el repartidor"/></FormField>
    <label className="switch-row compact"><input type="checkbox" {...register("vipOverride")}/><span/><b>Marcar como VIP</b></label>
   </div></FormSection></div>
  </ModalBody><ModalFooter><Button action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter></ModalForm></Dialog></div>
}
function CustomerDetail({customer,loading,error,retry,retrying,close}:{customer?:Customer;loading:boolean;error?:string;retry:()=>void;retrying:boolean;close:()=>void}){
 const{location}=useSession();const money=useMoney();
 const lastPurchase=customer?.lastPurchaseAt?formatRegionalDateTime(customer.lastPurchaseAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium"}):"—";
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal customer-detail modal-panel-in" role="dialog" aria-modal="true" aria-busy={loading}>

  {loading?<CustomerDetailSkeleton close={close}/>:<>
   <ModalHeader title="Detalle del cliente" icon="users" onClose={close} closeLabel="Cerrar"/>
   {error?<div className="customer-detail-body"><ErrorState title="No pudimos cargar el detalle" text={error} onRetry={retry} retrying={retrying}/></div>:customer&&<div className="customer-detail-body">
    <DetailHero title={customer.displayName} status={<Status active={customer.active}>{customer.active?"Activo":"Inactivo"}</Status>} meta={`${segmentLabels[customer.segment]}${customer.vipOverride?" · Cliente VIP":""}`}/>
    <DetailContact icon="user" name={customer.phone||"Sin teléfono registrado"} detail={customer.email||"Sin correo registrado"} actions={customer.phone&&<a className="button secondary" href={`tel:${customer.phone.replace(/\s+/g,"")}`} aria-label={`Llamar a ${customer.phone}`}><Icon name="phone" size={16}/><span>Llamar</span></a>}/>
    <DetailFacts label="Relación con el cliente" items={[
     {label:"Visitas",value:String(customer.visitCount)},
     {label:"Acumulado",value:money(customer.totalSpent)},
     {label:"Última compra",value:lastPurchase},
     {label:"Documento",value:customer.documentNumber?`${customer.documentType} ${customer.documentNumber}`:"No registrado"},
    ]}/>
    <section className="customer-detail-section">
     <h3 className="modal-section-title modal-section-rule">Dirección de entrega</h3>
     {customer.addresses?.length?customer.addresses.map(a=><div className="customer-detail-address" key={a.id}><div><b>{a.label}</b>{a.default&&customer.addresses!.length>1&&<span>Predeterminada</span>}</div><p>{a.address}{a.district?`, ${a.district}`:""}{a.city?`, ${a.city}`:""}</p>{a.latitude!=null&&a.longitude!=null&&<small className="customer-detail-located"><Icon name="mapPin" size={14}/>Ubicada en el mapa</small>}{a.reference&&<small>{a.reference}</small>}</div>):<p className="customer-detail-empty">No registrada.</p>}
    </section>
   </div>}
  </>}
</Dialog></div>
}

function CustomerDetailSkeleton({close}:{close:()=>void}){
 return <>
  <ModalHeader title={<span className="customer-detail-skeleton-copy" aria-hidden="true"><b/></span>} icon="users" onClose={close} initialFocusClose/>
  <div className="customer-detail-body customer-detail-skeleton-body" aria-label="Cargando detalle del cliente">
   <section className="customer-detail-summary customer-detail-skeleton-summary">
    {Array.from({length:4},(_,i)=><div key={i}><i/><b/></div>)}
    <span className="customer-detail-skeleton-block customer-detail-skeleton-status"/>
   </section>
   <section className="customer-detail-fields customer-detail-skeleton-fields">
    {Array.from({length:4},(_,i)=><div key={i}><i/><b/></div>)}
   </section>
   {Array.from({length:1},(_,i)=><section className="customer-detail-section customer-detail-skeleton-section" key={i}>
    <header><div><i/><b/></div></header>
    <div className="customer-detail-skeleton-lines"><i/><i/><i/></div>
   </section>)}
  </div>
 </>;
}
