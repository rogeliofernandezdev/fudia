"use client";
import "./payment-methods.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Input,PageHeader,Pagination,RowActionButton,Select,Status,Textarea} from "@/design-system/page-header";
import {ConfirmDialog} from "@/design-system/confirm-dialog";
import {Icon} from "@/design-system/icons";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import type {PaymentMethod,PaymentMethodDraft} from "../domain/payment-method";
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
   {query.isLoading?<PaymentMethodsLoading/>:query.isError?<div className="payment-method-empty"><span><Icon name="alert"/></span><b>No pudimos cargar los medios de pago</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>:!query.data?.items.length?<div className="payment-method-empty"><span><Icon name="payment"/></span><b>No hay medios de pago para mostrar</b><p>{q||status?"Cambia los filtros para ampliar la búsqueda.":"Crea el primer medio de pago de la empresa."}</p>{canManage&&!q&&!status&&<Button onClick={create}>Nuevo medio</Button>}</div>:<div className="table-wrap hover-scroll"><table className="payment-method-table"><thead><tr><th>MEDIO</th><th>VENTAS</th><th>GASTOS</th><th>CAJA</th><th>ORDEN</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{query.data.items.map((item,i)=><tr className={i%2?"alternate":""} key={item.code}><td><div className="payment-method-name"><div><span className={`row-icon r${i%3}`}><Icon name="payment" size={18}/></span><b>{item.name}</b><code>{item.code}</code></div><small>{item.description||"Sin descripción"}</small></div></td><td><Flag on={item.salesEnabled}>Disponible</Flag></td><td><Flag on={item.expensesEnabled}>Disponible</Flag></td><td><Flag on={item.affectsCash}>{item.affectsCash?"Impacta":"No impacta"}</Flag></td><td>{item.sortOrder}</td><td><Status tone={item.active?"green":"gray"}>{item.active?"Activo":"Inactivo"}</Status></td><td><div className="table-actions">{canManage&&<><RowActionButton action="edit" onClick={()=>edit(item)}/><RowActionButton action={item.active?"deactivate":"activate"} onClick={()=>setStatusTarget(item)}/></>}</div></td></tr>)}</tbody></table></div>}
   <Pagination page={page} size={size} total={query.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>
  </section>
  {draft&&<PaymentMethodDialog value={draft} editing={Boolean(editingCode)} busy={save.isPending} close={()=>{setDraft(null);setEditingCode(undefined)}} save={value=>save.mutate(value)}/>}
  <ConfirmDialog open={Boolean(statusTarget)} title={statusTarget?.active?"Desactivar medio de pago":"Activar medio de pago"} description={statusTarget?.active?`“${statusTarget?.name??""}” dejará de aparecer en nuevos cobros o gastos según su configuración. El historial se conserva.`:`“${statusTarget?.name??""}” volverá a estar disponible en los flujos configurados.`} tone={statusTarget?.active?"danger":"success"} confirmLabel={statusTarget?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>statusTarget&&changeStatus.mutate(statusTarget)}/>
 </>;
}

function Flag({on,children}:{on:boolean;children:React.ReactNode}){return <span className={`payment-method-flag ${on?"on":""}`}><i/>{children}</span>}

function PaymentMethodsLoading(){return <div className="table-skeleton" aria-label="Cargando medios de pago"><div className="sk-head"><i/><i/><i/><i/><i/></div>{Array.from({length:5},(_,i)=><div className="sk-row" key={i}><i className="sk-name"><span/><b/><small/></i><i/><i/><i/><i/></div>)}</div>}

function PaymentMethodDialog({value:initial,editing,busy,close,save}:{value:PaymentMethodDraft;editing:boolean;busy:boolean;close:()=>void;save:(value:PaymentMethodDraft)=>void}){
 const[value,setValue]=useState(initial);
 const codeValid=/^[a-z0-9][a-z0-9_-]{0,39}$/.test(value.code);
 const canSubmit=(editing||codeValid)&&value.name.trim().length>0&&(value.salesEnabled||value.expensesEnabled)&&value.sortOrder>=0&&value.sortOrder<=9999&&!busy;
 function submit(event:React.FormEvent){event.preventDefault();if(canSubmit)save({...value,code:value.code.trim().toLowerCase(),name:value.name.trim(),description:value.description.trim()})}
 return <div className="payment-method-modal-overlay"><section className="payment-method-modal" role="dialog" aria-modal="true" aria-labelledby="payment-method-title">
  <header><span><Icon name="payment"/></span><div><small>{editing?"EDITAR MEDIO":"NUEVO MEDIO"}</small><h2 id="payment-method-title">Configuración del medio de pago</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
  <form className="payment-method-form" onSubmit={submit}>
   <div className="payment-method-grid">
    <label>Código interno<Input required={!editing} readOnly={editing} maxLength={40} value={value.code} onChange={e=>setValue({...value,code:e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g,"")})} placeholder="ej. voucher"/><small>{editing?"El código es inmutable porque identifica pagos históricos.":"Usa minúsculas, números, guion o guion bajo."}</small></label>
    <label>Orden<Input required type="number" min={0} max={9999} value={value.sortOrder} onChange={e=>setValue({...value,sortOrder:Number(e.target.value)})}/><small>Menor número = aparece primero.</small></label>
    <label className="span-2">Nombre<Input required maxLength={80} value={value.name} onChange={e=>setValue({...value,name:e.target.value})} placeholder="Ej. Vale corporativo"/></label>
    <label className="span-2">Descripción<Textarea maxLength={180} rows={3} value={value.description} onChange={e=>setValue({...value,description:e.target.value})} placeholder="Describe cuándo debe utilizarse este medio."/></label>
   </div>
   <div className="payment-method-toggle-grid">
    <label className="payment-method-toggle"><input type="checkbox" checked={value.salesEnabled} onChange={e=>setValue({...value,salesEnabled:e.target.checked})}/><span><b>Ventas</b><small>Disponible al cobrar pedidos.</small></span></label>
    <label className="payment-method-toggle"><input type="checkbox" checked={value.expensesEnabled} onChange={e=>setValue({...value,expensesEnabled:e.target.checked})}/><span><b>Gastos</b><small>Disponible al registrar gastos.</small></span></label>
    <label className="payment-method-toggle"><input type="checkbox" checked={value.affectsCash} onChange={e=>setValue({...value,affectsCash:e.target.checked})}/><span><b>Impacta Caja</b><small>Se considera movimiento físico de efectivo en cobros y devoluciones.</small></span></label>
   </div>
   {!value.salesEnabled&&!value.expensesEnabled&&<div className="payment-method-warning">El medio debe estar habilitado al menos para Ventas o para Gastos.</div>}
   {!editing&&!codeValid&&value.code&&<div className="payment-method-warning">El código debe iniciar con letra o número y usar solo minúsculas, números, guiones o guion bajo.</div>}
   <footer><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="save" disabled={!canSubmit}>{busy?"Guardando…":"Guardar"}</Button></footer>
  </form>
 </section></div>
}
