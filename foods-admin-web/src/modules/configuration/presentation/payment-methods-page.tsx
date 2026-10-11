"use client";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import "./payment-methods.css";
import {useState} from "react";
import {useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,FormField,Input,PageHeader,Pagination,RowActionButton,SearchField,Select,Status,TableSkeleton,Textarea,ModalHeader,ModalFooter} from "@/design-system";
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
const columns=["MEDIO","VENTAS","GASTOS","CAJA","ORDEN","ESTADO","ACCIONES"];

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
 const filtered=Boolean(q||status);
 const items=query.data?.items??[];
 const actions=(item:PaymentMethod)=>canManage&&<><RowActionButton action="edit" onClick={()=>edit(item)}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado de ${item.name}`} onClick={()=>setStatusTarget(item)}/></>;
 return <><PageHeader title="Medios de pago" description="Administra el catálogo único utilizado por cobros, caja y gastos." action={canManage?<Button icon="plus" onClick={create}>Nuevo medio</Button>:undefined}/>
  <section className="panel management standardized-management payment-methods-panel">
   <div className="toolbar">
    <SearchField value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar por nombre, código o descripción..." label="Buscar medios de pago"/>
    <Select aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}>
     <option value="">Todos los estados</option>
     <option value="active">Activos</option>
     <option value="inactive">Inactivos</option>
    </Select>
   </div>
   {query.isLoading?<TableSkeleton columns={columns} label="Cargando medios de pago"/>:query.isError?<ErrorState title="No pudimos cargar los medios de pago" text={query.error.message} onRetry={()=>void query.refetch()} retrying={query.isFetching}/>:!items.length?<EmptyState icon="payment" title={filtered?"Sin coincidencias":"Aún no hay medios de pago"} text={filtered?"Ningún medio de pago coincide con los filtros.":"Crea el primer medio de pago de la empresa."} action={canManage&&!filtered?<Button icon="plus" onClick={create}>Nuevo medio</Button>:undefined}/>:<>
    <DataTable label="Medios de pago" containerClassName="hover-scroll" className="payment-method-table"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>{items.map((item,i)=><tr className={i%2?"alternate":""} key={item.code}><td><div className="payment-method-name"><div><span className={`row-icon r${i%3}`}><Icon name="payment" size={18}/></span><b>{item.name}</b><code>{item.code}</code></div><small>{item.description||"Sin descripción"}</small></div></td><td><Availability on={item.salesEnabled}/></td><td><Availability on={item.expensesEnabled}/></td><td><CashImpact on={item.affectsCash}/></td><td>{item.sortOrder}</td><td><Status active={item.active}>{item.active?"Activo":"Inactivo"}</Status></td><td><div className="table-actions">{actions(item)}</div></td></tr>)}</tbody></DataTable>
    <div className="management-cards">{items.map(item=><article key={item.code}><header><span className="row-icon"><Icon name="payment"/></span><div><b>{item.name}</b><small>{item.code}</small></div><Status active={item.active}>{item.active?"Activo":"Inactivo"}</Status></header><dl><div><dt>Ventas</dt><dd><Availability on={item.salesEnabled}/></dd></div><div><dt>Gastos</dt><dd><Availability on={item.expensesEnabled}/></dd></div><div><dt>Caja</dt><dd><CashImpact on={item.affectsCash}/></dd></div><div><dt>Orden</dt><dd>{item.sortOrder}</dd></div></dl>{canManage&&<footer>{actions(item)}</footer>}</article>)}</div>
   </>}
   <Pagination page={page} size={size} total={query.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>
  </section>
  {draft&&<PaymentMethodDialog value={draft} editing={Boolean(editingCode)} busy={save.isPending} close={()=>{setDraft(null);setEditingCode(undefined)}} save={value=>save.mutate(value)}/>}
  <ConfirmDialog open={Boolean(statusTarget)} title={statusTarget?.active?"Desactivar medio de pago":"Activar medio de pago"} description={statusTarget?.active?`“${statusTarget?.name??""}” dejará de aparecer en nuevos cobros o gastos según su configuración. El historial se conserva.`:`“${statusTarget?.name??""}” volverá a estar disponible en los flujos configurados.`} tone={statusTarget?.active?"danger":"success"} confirmLabel={statusTarget?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>statusTarget&&changeStatus.mutate(statusTarget)}/>
 </>;
}

function Availability({on}:{on:boolean}){return <Status tone={on?"green":"gray"}>{on?"Habilitado":"No habilitado"}</Status>}
function CashImpact({on}:{on:boolean}){return <Status tone={on?"blue":"gray"}>{on?"Impacta":"No impacta"}</Status>}

function PaymentMethodDialog({value:initial,editing,busy,close,save}:{value:PaymentMethodDraft;editing:boolean;busy:boolean;close:()=>void;save:(value:PaymentMethodDraft)=>void}){
 const{register,handleSubmit,setValue,formState:{errors}}=useForm<PaymentMethodDraft>({defaultValues:initial,resolver:paymentMethodResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const submit=handleSubmit(value=>save({...value,code:value.code.trim().toLowerCase(),name:value.name.trim(),description:value.description.trim(),sortOrder:Number(value.sortOrder)}));
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal payment-method-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="payment-method-title">
  <ModalHeader title="Configuración del medio de pago" titleId="payment-method-title" icon="payment" subtitle={editing?"EDITAR MEDIO":"NUEVO MEDIO"} onClose={close} closeLabel="Cerrar"/>
  <form onSubmit={submit} noValidate><div className="payment-method-form-body">
   <div className="form-grid">
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
  </div><ModalFooter><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter></form>
 </Dialog></div>
}
