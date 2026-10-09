"use client";
import {Dialog} from "@/design-system/dialog";
import "./payment-methods.css";
import {useState} from "react";
import {useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Input,PageHeader,Pagination,RowActionButton,Select,Status,Textarea} from "@/design-system";
import {ConfirmDialog} from "@/design-system/confirm-dialog";
import {Icon} from "@/design-system/icons";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import type {PaymentMethod,PaymentMethodDraft} from "../domain/payment-method";
import {paymentMethodResolver} from "../domain/payment-method-schema";
import {listPaymentMethods,savePaymentMethod,setPaymentMethodActive} from "../infrastructure/payment-methods-api";

const blank:PaymentMethodDraft={
 code:"",
 name:"",
 description:"",
 salesEnabled:true,
 expensesEnabled:true,
 affectsCash:false,
 sortOrder:100,
};

export function PaymentMethodsPage(){
 const{can}=useSession();
 const{notify}=useFeedback();
 const client=useQueryClient();
 const canManage=can("organizations.manage");
 const[q,setQ]=useState("");
 const[status,setStatus]=useState("");
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(20);
 const[draft,setDraft]=useState<PaymentMethodDraft|null>(null);
 const[editingCode,setEditingCode]=useState<string|undefined>();
 const[statusTarget,setStatusTarget]=useState<PaymentMethod|null>(null);
 const query=useQuery({queryKey:["payment-methods-admin",q,status,page,size],queryFn:()=>listPaymentMethods({q,status,page,pageSize:size})});
 const save=useMutation({
  mutationFn:(value:PaymentMethodDraft)=>savePaymentMethod(value,editingCode),
  onSuccess:()=>{setDraft(null);setEditingCode(undefined);void client.invalidateQueries({queryKey:["payment-methods-admin"]});notify({tone:"success",title:"Medio de pago guardado",message:"La configuración ya está disponible para los flujos correspondientes."})},
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const changeStatus=useMutation({
  mutationFn:(item:PaymentMethod)=>setPaymentMethodActive(item.code,!item.active),
  onSuccess:()=>{setStatusTarget(null);void client.invalidateQueries({queryKey:["payment-methods-admin"]});notify({tone:"success",title:"Estado actualizado",message:"El catálogo operativo se actualizó correctamente."})},
  onError:e=>{setStatusTarget(null);notify({tone:"danger",title:"No se pudo actualizar",message:e.message})},
 });
 function create(){setEditingCode(undefined);setDraft({...blank})}
 function edit(item:PaymentMethod){setEditingCode(item.code);setDraft({code:item.code,name:item.name,description:item.description,salesEnabled:item.salesEnabled,expensesEnabled:item.expensesEnabled,affectsCash:item.affectsCash,sortOrder:item.sortOrder})}
 return <><PageHeader eyebrow="CONFIGURACIÓN" title="Medios de pago" description="Administra el catálogo único utilizado por cobros, caja y gastos." action={canManage?<Button icon="plus" onClick={create}>Nuevo medio</Button>:undefined}/>
  <section className="panel management catalog-panel">
   <div className="payment-method-toolbar">
    <label className="payment-method-search">
     <Icon name="search" size={18}/>
     <Input aria-label="Buscar medios de pago" value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar por nombre, código o descripción..."/>
    </label>
    <Select className="payment-method-status-filter" aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}>
     <option value="">Todos los estados</option>
     <option value="active">Activos</option>
     <option value="inactive">Inactivos</option>
    </Select>
   </div>
   {query.isLoading?<PaymentMethodsLoading/>:query.isError?<div className="payment-method-empty"><span><Icon name="alert"/></span><b>No pudimos cargar los medios de pago</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>:!query.data?.items.length?<div className="payment-method-empty"><span><Icon name="payment"/></span><b>No hay medios de pago para mostrar</b><p>{q||status?"Cambia los filtros para ampliar la búsqueda.":"Crea el primer medio de pago de la empresa."}</p>{canManage&&!q&&!status&&<Button onClick={create}>Nuevo medio</Button>}</div>:<div className="table-wrap hover-scroll"><table className="payment-method-table"><thead><tr><th>MEDIO</th><th>VENTAS</th><th>GASTOS</th><th>CAJA</th><th>ORDEN</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{query.data.items.map((item,i)=><tr className={i%2?"alternate":""} key={item.code}><td><div className="payment-method-name"><div><span className={`row-icon r${i%3}`}><Icon name="payment" size={18}/></span><b>{item.name}</b><code>{item.code}</code></div><small>{item.description||"Sin descripción"}</small></div></td><td><Flag on={item.salesEnabled}>Disponible</Flag></td><td><Flag on={item.expensesEnabled}>Disponible</Flag></td><td><Flag on={item.affectsCash}>{item.affectsCash?"Impacta":"No impacta"}</Flag></td><td>{item.sortOrder}</td><td><Status active={item.active}>{item.active?"Activo":"Inactivo"}</Status></td><td><div className="table-actions">{canManage&&<><RowActionButton action="edit" onClick={()=>edit(item)}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado de ${item.name}`} onClick={()=>setStatusTarget(item)}/></>}</div></td></tr>)}</tbody></table></div>}
   <Pagination page={page} size={size} total={query.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>
  </section>
  {draft&&<PaymentMethodDialog value={draft} editing={Boolean(editingCode)} busy={save.isPending} close={()=>{setDraft(null);setEditingCode(undefined)}} save={value=>save.mutate(value)}/>}
  <ConfirmDialog open={Boolean(statusTarget)} title={statusTarget?.active?"Desactivar medio de pago":"Activar medio de pago"} description={statusTarget?.active?`“${statusTarget?.name??""}” dejará de aparecer en nuevos cobros o gastos según su configuración. El historial se conserva.`:`“${statusTarget?.name??""}” volverá a estar disponible en los flujos configurados.`} tone={statusTarget?.active?"danger":"success"} confirmLabel={statusTarget?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>statusTarget&&changeStatus.mutate(statusTarget)}/>
 </>;
}

function Flag({on,children}:{on:boolean;children:React.ReactNode}){return <span className={`payment-method-flag ${on?"on":""}`}><i/>{children}</span>}

function PaymentMethodsLoading(){return <div className="table-skeleton" aria-label="Cargando medios de pago"><div className="sk-head"><i/><i/><i/><i/><i/></div>{Array.from({length:5},(_,i)=><div className="sk-row" key={i}><i className="sk-name"><span/><b/><small/></i><i/><i/><i/><i/></div>)}</div>}

function PaymentMethodDialog({value:initial,editing,busy,close,save}:{value:PaymentMethodDraft;editing:boolean;busy:boolean;close:()=>void;save:(value:PaymentMethodDraft)=>void}){
 const{register,handleSubmit,setValue,formState:{errors}}=useForm<PaymentMethodDraft>({defaultValues:initial,resolver:paymentMethodResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const submit=handleSubmit(value=>save({...value,code:value.code.trim().toLowerCase(),name:value.name.trim(),description:value.description.trim(),sortOrder:Number(value.sortOrder)}));
 return <div className="payment-method-modal-overlay"><Dialog onResponseClose={close} className="payment-method-modal" role="dialog" aria-modal="true" aria-labelledby="payment-method-title">
  <header><span><Icon name="payment"/></span><div><small>{editing?"EDITAR MEDIO":"NUEVO MEDIO"}</small><h2 id="payment-method-title">Configuración del medio de pago</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
  <form className="payment-method-form" onSubmit={submit} noValidate>
   <div className="payment-method-grid">
    <FormField label="Código interno" help={editing?"El código no se puede cambiar.":"Minúsculas, números, guion o guion bajo."} error={errors.code?.message}><Input readOnly={editing} maxLength={40} {...register("code",{onChange:e=>setValue("code",e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g,""))})} placeholder="ej. voucher"/></FormField>
    <FormField label="Orden" help="Menor número aparece primero." error={errors.sortOrder?.message}><Input type="number" inputMode="numeric" min={0} max={9999} {...register("sortOrder")}/></FormField>
    <FormField className="span-2" label="Nombre" error={errors.name?.message}><Input maxLength={80} {...register("name")} placeholder="Ej. Vale corporativo"/></FormField>
    <FormField className="span-2" label="Descripción" optional error={errors.description?.message}><Textarea maxLength={180} rows={3} {...register("description")} placeholder="Describe cuándo debe utilizarse este medio."/></FormField>
   </div>
   <div className="payment-method-toggle-grid">
    <label className="payment-method-toggle"><input type="checkbox" {...register("salesEnabled")}/><span><b>Ventas</b><small>Disponible al cobrar pedidos.</small></span></label>
    <label className="payment-method-toggle"><input type="checkbox" {...register("expensesEnabled")}/><span><b>Gastos</b><small>Disponible al registrar gastos.</small></span></label>
    <label className="payment-method-toggle"><input type="checkbox" {...register("affectsCash")}/><span><b>Impacta Caja</b><small>Se considera movimiento físico de efectivo en cobros y devoluciones.</small></span></label>
   </div>
   {errors.salesEnabled?.message&&<div className="payment-method-warning" role="alert">{errors.salesEnabled.message}</div>}
   <footer><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button></footer>
  </form>
 </Dialog></div>
}
