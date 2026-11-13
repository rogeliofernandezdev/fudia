"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import {Button,ConfirmDialog,EmptyState,ErrorState,FormField,Icon,IconButton,Input,PageHeader,RowActionButton,Status,TableSkeleton,ModalHeader,ModalFooter,ModalForm,ModalBody} from "@/design-system";
import {useFeedback} from "@/providers";
import type {DeliveryPartner,DeliveryPartnerDraft} from "../domain/types";
import {listDeliveryPartners,reorderDeliveryPartners,saveDeliveryPartner} from "../infrastructure/platform-api";
import "./platform-delivery-partners.css";
import {DeliveryCoverageFields} from "./delivery-coverage-fields";

const columns=["Orden","Central","WhatsApp","Cobertura","Estado","Acciones"];
const blank:DeliveryPartnerDraft={name:"",whatsappPhone:"51",coverageCodes:[],coverageLabels:[]};
const queryKey=["platform-delivery-partners"];

/** WhatsApp con código de país, agrupado para leerlo: +51 987 654 321. */
export function formatPartnerPhone(digits:string){
 const match=/^(51)(\d{3})(\d{3})(\d{3})$/.exec(digits);
 return match?`+${match[1]} ${match[2]} ${match[3]} ${match[4]}`:`+${digits}`;
}

/**
 * Centrales de Delivery Fudia: la plataforma registra las centrales de reparto
 * y su orden global y su cobertura; cada pedido de una empresa con Delivery Fudia se ofrece a
 * la 1.ª y, si rechaza o no responde, a la siguiente.
 */
export function PlatformDeliveryPartnersPage(){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const partners=useQuery({queryKey,queryFn:listDeliveryPartners});
 const[draft,setDraft]=useState<DeliveryPartnerDraft|null>(null);
 const[editingId,setEditingId]=useState<string|undefined>();
 const[target,setTarget]=useState<DeliveryPartner|null>(null);
 const close=()=>{setDraft(null);setEditingId(undefined)};
 const save=useMutation({
  mutationFn:({value,id}:{value:DeliveryPartnerDraft;id?:string})=>saveDeliveryPartner(value,id),
  onSuccess:(_,{id})=>{
   close();
   void client.invalidateQueries({queryKey});
   notify({tone:"success",title:id?"Central actualizada":"Central registrada",message:id?"La cobertura y el estado se aplican a las ofertas pendientes.":"Quedó al final del orden; súbela si debe recibir antes."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const toggle=useMutation({
  mutationFn:(item:DeliveryPartner)=>saveDeliveryPartner({name:item.name,whatsappPhone:item.whatsappPhone,coverageCodes:item.coverageCodes,active:!item.active},item.id),
  onSuccess:(_,item)=>{
   setTarget(null);
   void client.invalidateQueries({queryKey});
   notify({tone:"success",title:item.active?"Central desactivada":"Central activada",message:item.active?"Las ofertas pendientes pasan a la siguiente central con cobertura.":"Vuelve a recibir pedidos en su lugar del orden."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message}),
 });
 const reorder=useMutation({
  mutationFn:reorderDeliveryPartners,
  onSuccess:data=>client.setQueryData(queryKey,data),
  onError:e=>{void client.invalidateQueries({queryKey});notify({tone:"danger",title:"No se pudo cambiar el orden",message:e.message});},
 });
 const move=(index:number,offset:-1|1)=>{
  const ids=items.map(item=>item.id);
  const target=index+offset;
  if(target<0||target>=ids.length)return;
  [ids[index],ids[target]]=[ids[target],ids[index]];
  reorder.mutate(ids);
 };
 const start=()=>{setEditingId(undefined);setDraft({...blank})};
 const edit=(item:DeliveryPartner)=>{setEditingId(item.id);setDraft({name:item.name,whatsappPhone:item.whatsappPhone,coverageCodes:item.coverageCodes,coverageLabels:item.coverageLabels})};
 const items=partners.data?.items??[];
 const status=(item:DeliveryPartner)=><Status active={item.active}>{item.active?"Activa":"Inactiva"}</Status>;
 const order=(index:number)=><div className="partner-order"><b>{index+1}</b><IconButton icon="chevron" className="partner-order-up" label="Subir en el orden" disabled={index===0||reorder.isPending} onClick={()=>move(index,-1)}/><IconButton icon="chevron" className="partner-order-down" label="Bajar en el orden" disabled={index===items.length-1||reorder.isPending} onClick={()=>move(index,1)}/></div>;
 const actions=(item:DeliveryPartner)=><><RowActionButton action="edit" label={`Editar central ${item.name}`} onClick={()=>edit(item)}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado de la central ${item.name}`} onClick={()=>setTarget(item)}/></>;

 return <>
  <PageHeader title="Centrales de delivery" description="Reciben por WhatsApp los pedidos de las empresas con Delivery Fudia, según el departamento, provincia y distrito del local, en este orden." action={<Button icon="plus" onClick={start}>Nueva central</Button>}/>
  <section className="panel management standardized-management platform-partners-panel">
   {partners.isLoading?<TableSkeleton columns={columns} rows={4} label="Cargando centrales de delivery"/>:
    partners.isError?<ErrorState title="No pudimos cargar las centrales" text={partners.error.message} retrying={partners.isFetching} onRetry={()=>void partners.refetch()}/>:
    !items.length?<EmptyState icon="motorbike" title="Aún no hay centrales" text="Registra la primera central de reparto para ofrecer Delivery Fudia a las empresas." action={<Button icon="plus" onClick={start}>Nueva central</Button>}/>:
    <><DataTable label="Centrales de delivery" containerClassName="hover-scroll"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>
     {items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}><td>{order(index)}</td><td><span className={`row-icon r${index%3}`}><Icon name="motorbike" size={18}/></span><b>{item.name}</b></td><td>{formatPartnerPhone(item.whatsappPhone)}</td><td>{item.coverageLabels.length?item.coverageLabels.map(label=><small key={label}>{label}</small>):"Sin cobertura"}</td><td>{status(item)}</td><td><div className="table-actions">{actions(item)}</div></td></tr>)}
    </tbody></DataTable>
    <div className="management-cards">{items.map((item,index)=><article key={item.id}><header><span className="row-icon r0"><Icon name="motorbike"/></span><div><b>{item.name}</b><small>{formatPartnerPhone(item.whatsappPhone)}</small><small>{item.coverageLabels.join("; ")||"Sin cobertura"}</small></div>{status(item)}</header><footer>{order(index)}{actions(item)}</footer></article>)}</div></>}
  </section>

  {draft&&<div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal platform-partner-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="delivery-partner-title" aria-busy={save.isPending}><ModalHeader title={editingId?"Editar central":"Nueva central"} titleId="delivery-partner-title" onClose={close} busy={save.isPending} closeLabel="Cerrar"/>
   <ModalForm onSubmit={event=>{event.preventDefault();save.mutate({value:draft,id:editingId})}}>
    <ModalBody><div className="form-sections-flow"><div className="form-grid platform-partner-fields">
     <FormField className="span-2" label="Nombre"><Input value={draft.name} onChange={e=>setDraft({...draft,name:e.target.value})} placeholder="Central Norte" autoFocus required/></FormField>
     <FormField className="span-2" label="WhatsApp" help="Con código de país. Aquí llegan los pedidos con Aceptar y Rechazar."><Input value={draft.whatsappPhone} onChange={e=>setDraft({...draft,whatsappPhone:e.target.value})} inputMode="tel" placeholder="51 987 654 321" required/></FormField>
    </div><DeliveryCoverageFields codes={draft.coverageCodes} labels={draft.coverageLabels??[]} disabled={save.isPending} onChange={(coverageCodes,coverageLabels)=>setDraft({...draft,coverageCodes,coverageLabels})}/></div></ModalBody>
    <ModalFooter><Button type="button" action="cancel" disabled={save.isPending} onClick={close}>Cancelar</Button><Button type="submit" disabled={save.isPending||!draft.coverageCodes.length} loading={save.isPending}>{save.isPending?"Guardando…":"Guardar"}</Button></ModalFooter>
   </ModalForm>
  </Dialog></div>}
  <ConfirmDialog open={Boolean(target)} title={target?.active?"Desactivar central":"Activar central"} subject={target?{label:"Central",name:target.name}:undefined} description={target?.active?"Las ofertas pendientes pasarán a la siguiente central con cobertura. Los pedidos ya aceptados conservan su central.":"Volverá a recibir pedidos en su lugar del orden."} tone={target?.active?"danger":"success"} confirmLabel={target?.active?"Desactivar":"Activar"} pending={toggle.isPending} onCancel={()=>setTarget(null)} onConfirm={()=>target&&toggle.mutate(target)}/>
 </>;
}
