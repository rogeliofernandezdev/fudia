"use client";
import "./purchases.css";
import {useState} from "react";
import {useFieldArray,useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status,Textarea} from "@/design-system";
import {useFeedback,useSession,useSettings} from "@/providers";
import {formatRegionalCalendarDate,formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {purchaseOrderResolver,supplierResolver} from "../domain/purchase-schema";
import {PurchaseItemDialog} from "./purchase-item-dialog";
import {PurchaseReceiptDialog} from "./purchase-receipt-dialog";
import type {PurchaseInventoryOption,PurchaseOrder,PurchaseOrderDraft,PurchaseOrderSummary,PurchaseStatus,PurchaseTab,Supplier,SupplierDraft} from "../domain/types";
import {createPurchaseInventoryItem,getPurchaseOrder,listPurchaseInventory,listPurchaseItemCategories,listPurchaseOrders,listSuppliers,receivePurchaseOrder,savePurchaseOrder,saveSupplier,setPurchaseOrderStatus,setSupplierActive} from "../infrastructure/purchases-api";

const statusMeta:Record<PurchaseStatus,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={
  draft:{label:"Borrador",tone:"gray"},
  pending_approval:{label:"Por aprobar",tone:"orange"},
  approved:{label:"Aprobada",tone:"blue"},
  partially_received:{label:"Recepción parcial",tone:"orange"},
  received:{label:"Recibida",tone:"green"},
  cancelled:{label:"Cancelada",tone:"gray"},
};
const emptyOrder:PurchaseOrderDraft={id:"",supplierId:"",expectedAt:"",notes:"",items:[{inventoryItemId:"",presentationId:"",quantity:"1",unitCost:""}]};
const emptySupplier:SupplierDraft={id:"",taxId:"",name:"",email:"",phone:""};

export function PurchasesPage(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const settings=useSettings();
  const canManage=can("purchases.manage");
  const canReceive=can("purchases.receive");
  const[tab,setTab]=useState<PurchaseTab>("orders");
  const[q,setQ]=useState("");
  const[status,setStatus]=useState("");
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[orderDraft,setOrderDraft]=useState<PurchaseOrderDraft|null>(null);
  const[orderDraftLoading,setOrderDraftLoading]=useState(false);
  const[detailId,setDetailId]=useState<string|null>(null);
  const[supplierDraft,setSupplierDraft]=useState<SupplierDraft|null>(null);
  const[supplierTarget,setSupplierTarget]=useState<Supplier|null>(null);
  const[cancelTarget,setCancelTarget]=useState<PurchaseOrderSummary|PurchaseOrder|null>(null);
  const[receiptOrder,setReceiptOrder]=useState<PurchaseOrder|null>(null);

  const orders=useQuery({queryKey:["purchase-orders",q,status,page,size],queryFn:()=>listPurchaseOrders({q,status,page,pageSize:size}),enabled:tab==="orders"});
  const suppliers=useQuery({queryKey:["suppliers",q,status,page,size],queryFn:()=>listSuppliers({q,status,page,pageSize:size}),enabled:tab==="suppliers"});
  const activeSuppliers=useQuery({queryKey:["suppliers","active","purchase-editor"],queryFn:()=>listSuppliers({status:"active",page:1,pageSize:100}),enabled:Boolean(orderDraft),staleTime:30000});
  const inventory=useQuery({queryKey:["purchase-inventory"],queryFn:listPurchaseInventory,enabled:Boolean(orderDraft),staleTime:30000});
  const detail=useQuery({queryKey:["purchase-order",detailId],queryFn:()=>getPurchaseOrder(detailId!),enabled:Boolean(detailId)});

  const saveOrder=useMutation({mutationFn:savePurchaseOrder,onSuccess:result=>{setOrderDraft(null);void qc.invalidateQueries({queryKey:["purchase-orders"]});void qc.invalidateQueries({queryKey:["dashboard"]});notify({tone:"success",title:"Orden guardada",message:`${result.number} quedó en borrador y lista para revisión.`})},onError:error=>notify({tone:"danger",title:"No se pudo guardar la orden",message:error.message})});
  const transition=useMutation({mutationFn:({id,next}:{id:string;next:"draft"|"pending_approval"|"approved"|"cancelled"})=>setPurchaseOrderStatus(id,next),onSuccess:(_,variables)=>{void qc.invalidateQueries({queryKey:["purchase-orders"]});void qc.invalidateQueries({queryKey:["purchase-order"]});void qc.invalidateQueries({queryKey:["dashboard"]});if(variables.next==="cancelled"){setCancelTarget(null)}notify({tone:"success",title:"Estado actualizado",message:"La orden de compra quedó actualizada."})},onError:error=>notify({tone:"danger",title:"No se pudo actualizar la orden",message:error.message})});
  const receive=useMutation({mutationFn:receivePurchaseOrder,onSuccess:result=>{setReceiptOrder(null);void qc.invalidateQueries({queryKey:["purchase-orders"]});void qc.invalidateQueries({queryKey:["purchase-order"]});void qc.invalidateQueries({queryKey:["purchase-inventory"]});void qc.invalidateQueries({queryKey:["inventory"]});void qc.invalidateQueries({queryKey:["inventory-products"]});void qc.invalidateQueries({queryKey:["inventory-movements"]});void qc.invalidateQueries({queryKey:["dashboard"]});notify({tone:"success",title:result.status==="received"?"Recepción completada":"Recepción parcial registrada",message:`${result.code} quedó vinculada a ${result.number} y actualizó Inventario y Kárdex.`})},onError:error=>notify({tone:"danger",title:"No se pudo registrar la recepción",message:error.message})});
  const supplierSave=useMutation({mutationFn:saveSupplier,onSuccess:()=>{setSupplierDraft(null);void qc.invalidateQueries({queryKey:["suppliers"]});notify({tone:"success",title:"Proveedor guardado",message:"El directorio de proveedores quedó actualizado."})},onError:error=>notify({tone:"danger",title:"No se pudo guardar el proveedor",message:error.message})});
  const supplierStatus=useMutation({mutationFn:(supplier:Supplier)=>setSupplierActive(supplier.id,!supplier.active),onSuccess:()=>{setSupplierTarget(null);void qc.invalidateQueries({queryKey:["suppliers"]});notify({tone:"success",title:"Estado actualizado",message:"El proveedor conserva su historial de compras."})},onError:error=>notify({tone:"danger",title:"No se pudo actualizar el proveedor",message:error.message})});

  async function editOrder(id:string){
    setOrderDraftLoading(true);
    setOrderDraft({...emptyOrder,id});
    try{
      const order=await getPurchaseOrder(id);
      setOrderDraft({
        id:order.id,
        supplierId:order.supplierId,
        expectedAt:order.expectedAt??"",
        notes:order.notes,
        items:order.items.map(item=>({inventoryItemId:item.inventoryItemId,presentationId:item.presentationId,quantity:item.quantity,unitCost:item.unitCost})),
      });
    }catch(error){
      setOrderDraft(null);
      notify({tone:"danger",title:"No se pudo cargar la orden",message:(error as Error).message});
    }finally{setOrderDraftLoading(false)}
  }

  function changeTab(next:PurchaseTab){
    setTab(next);setQ("");setStatus("");setPage(1);
  }

  const orderItems=orders.data?.items??[];
  const supplierItems=suppliers.data?.items??[];
  const headerAction=tab==="orders"
    ?canManage?<Button icon="plus" onClick={()=>setOrderDraft({...emptyOrder})}>Nueva orden</Button>:undefined
    :canManage?<Button icon="plus" onClick={()=>setSupplierDraft({...emptySupplier})}>Nuevo proveedor</Button>:undefined;

  return <>
    <PageHeader eyebrow="ABASTECIMIENTO" title="Compras" description="Ordena, aprueba y recibe mercadería con trazabilidad directa hacia Inventario y Kárdex." action={headerAction}/>
    <section className="panel standardized-management purchases-panel">
      <div className="purchases-tabs" role="tablist" aria-label="Compras">
        <button type="button" role="tab" aria-selected={tab==="orders"} className={tab==="orders"?"active":""} onClick={()=>changeTab("orders")}><Icon name="receipt" size={17}/><span>Órdenes</span>{orders.data&&<b>{orders.data.total}</b>}</button>
        <button type="button" role="tab" aria-selected={tab==="suppliers"} className={tab==="suppliers"?"active":""} onClick={()=>changeTab("suppliers")}><Icon name="truck" size={17}/><span>Proveedores</span>{suppliers.data&&<b>{suppliers.data.total}</b>}</button>
      </div>
      <div className="purchases-toolbar">
        <label className="purchases-search"><Icon name="search" size={18}/><Input value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder={tab==="orders"?"Buscar por orden o proveedor...":"Buscar proveedor o RUC..."}/></label>
        <Select aria-label="Filtrar por estado" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
          {tab==="orders"?<><option value="">Todos los estados</option>{Object.entries(statusMeta).map(([value,meta])=><option value={value} key={value}>{meta.label}</option>)}</>:<><option value="">Todos los estados</option><option value="active">Activos</option><option value="inactive">Inactivos</option></>}
        </Select>
        <p><Icon name="store" size={15}/>{tab==="orders"?"Órdenes del local activo; la recepción actualiza existencias automáticamente.":"Proveedores compartidos por la empresa para abastecer cualquiera de sus locales."}</p>
      </div>

      {tab==="orders"?<>
        {orders.isLoading?<PurchaseTableSkeleton/>:orders.isError?<PurchaseState icon="alert" title="No pudimos cargar las compras" text={orders.error.message} action={()=>orders.refetch()}/>:!orderItems.length?<PurchaseState icon="receipt" title={q||status?"Sin coincidencias":"Aún no hay órdenes de compra"} text={q||status?"Ajusta la búsqueda o los filtros.":canManage?"Crea una orden y agrega los artículos que necesitas abastecer.":"No hay órdenes registradas para este local."} action={canManage&&!q&&!status?()=>setOrderDraft({...emptyOrder}):undefined}/>:<>
          <div className="table-wrap hover-scroll purchases-table-wrap"><table><thead><tr><th>ORDEN</th><th>PROVEEDOR</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{orderItems.map((order,index)=>{const meta=statusMeta[order.status];return <tr className={index%2?"alternate":""} key={order.id}>
            <td><span className={`row-icon r${index%3}`}><Icon name="receipt" size={18}/></span><b>{order.number}</b><small>{order.itemCount} {order.itemCount===1?"artículo":"artículos"}</small></td>
            <td><b>{order.supplierName}</b>{order.expectedAt&&<small>Entrega esperada: {formatRegionalCalendarDate(order.expectedAt,location?.country,{dateStyle:"medium"})}</small>}</td>
            <td>{formatRegionalDateTime(order.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium"})}</td>
            <td><b className="purchase-money">{formatMoney(Number(order.total),settings,location?.country)}</b></td>
            <td><Status tone={meta.tone}>{meta.label}</Status></td>
            <td><div className="table-actions"><RowActionButton action="view" onClick={()=>setDetailId(order.id)}/>{canManage&&order.status==="draft"&&<RowActionButton action="edit" onClick={()=>void editOrder(order.id)}/>}</div></td>
          </tr>})}</tbody></table></div>
          <div className="management-cards purchases-cards">{orderItems.map(order=>{const meta=statusMeta[order.status];return <article key={order.id}><header><span className="row-icon r0"><Icon name="receipt"/></span><div><b>{order.number}</b><small>{order.supplierName}</small></div><Status tone={meta.tone}>{meta.label}</Status></header><dl><div><dt>Artículos</dt><dd>{order.itemCount}</dd></div><div><dt>Total</dt><dd>{formatMoney(Number(order.total),settings,location?.country)}</dd></div><div><dt>Creada</dt><dd>{formatRegionalDateTime(order.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"short"})}</dd></div></dl><footer><RowActionButton action="view" onClick={()=>setDetailId(order.id)}/>{canManage&&order.status==="draft"&&<RowActionButton action="edit" onClick={()=>void editOrder(order.id)}/>}</footer></article>})}</div>
        </>}
        {!orders.isLoading&&!orders.isError&&<Pagination page={page} size={size} total={orders.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
      </>:<>
        {suppliers.isLoading?<SupplierTableSkeleton/>:suppliers.isError?<PurchaseState icon="alert" title="No pudimos cargar los proveedores" text={suppliers.error.message} action={()=>suppliers.refetch()}/>:!supplierItems.length?<PurchaseState icon="truck" title={q||status?"Sin coincidencias":"Aún no hay proveedores"} text={q||status?"Ajusta la búsqueda o los filtros.":canManage?"Registra el primer proveedor para empezar a crear órdenes.":"No hay proveedores registrados."} action={canManage&&!q&&!status?()=>setSupplierDraft({...emptySupplier}):undefined}/>:<>
          <div className="table-wrap hover-scroll purchases-table-wrap"><table><thead><tr><th>PROVEEDOR</th><th>RUC</th><th>CONTACTO</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{supplierItems.map((supplier,index)=><tr className={index%2?"alternate":""} key={supplier.id}>
            <td><span className={`row-icon r${index%3}`}><Icon name="truck" size={18}/></span><b>{supplier.name}</b></td><td>{supplier.taxId||"—"}</td><td>{supplier.phone||supplier.email||"Sin contacto"}{supplier.phone&&supplier.email&&<small>{supplier.email}</small>}</td><td><Status tone={supplier.active?"green":"gray"}>{supplier.active?"Activo":"Inactivo"}</Status></td><td><div className="table-actions">{canManage&&<><RowActionButton action="edit" onClick={()=>setSupplierDraft({id:supplier.id,taxId:supplier.taxId,name:supplier.name,email:supplier.email,phone:supplier.phone})}/><RowActionButton action={supplier.active?"deactivate":"activate"} onClick={()=>setSupplierTarget(supplier)}/></>}</div></td>
          </tr>)}</tbody></table></div>
          <div className="management-cards purchases-cards">{supplierItems.map(supplier=><article key={supplier.id}><header><span className="row-icon r1"><Icon name="truck"/></span><div><b>{supplier.name}</b><small>{supplier.taxId||"Sin RUC"}</small></div><Status tone={supplier.active?"green":"gray"}>{supplier.active?"Activo":"Inactivo"}</Status></header><dl><div><dt>Teléfono</dt><dd>{supplier.phone||"—"}</dd></div><div><dt>Correo</dt><dd>{supplier.email||"—"}</dd></div></dl>{canManage&&<footer><RowActionButton action="edit" onClick={()=>setSupplierDraft({id:supplier.id,taxId:supplier.taxId,name:supplier.name,email:supplier.email,phone:supplier.phone})}/><RowActionButton action={supplier.active?"deactivate":"activate"} onClick={()=>setSupplierTarget(supplier)}/></footer>}</article>)}</div>
        </>}
        {!suppliers.isLoading&&!suppliers.isError&&<Pagination page={page} size={size} total={suppliers.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
      </>}
    </section>

    {orderDraft&&(orderDraftLoading||activeSuppliers.isLoading||inventory.isLoading?<RemoteModalSkeleton className="purchase-order-modal" label="Cargando datos de compra" rows={7} close={()=>{setOrderDraft(null);setOrderDraftLoading(false)}}/>:activeSuppliers.isError||inventory.isError?<PurchaseEditorError message={(activeSuppliers.error??inventory.error)?.message??"No pudimos cargar los datos necesarios."} close={()=>setOrderDraft(null)} retry={()=>{void activeSuppliers.refetch();void inventory.refetch()}}/>:<PurchaseOrderDialog initial={orderDraft} suppliers={activeSuppliers.data?.items??[]} inventory={inventory.data?.items??[]} currencySymbol={settings.currencySymbol} busy={saveOrder.isPending} close={()=>setOrderDraft(null)} save={draft=>saveOrder.mutate(draft)}/>)}
    {supplierDraft&&<SupplierDialog initial={supplierDraft} busy={supplierSave.isPending} close={()=>setSupplierDraft(null)} save={draft=>supplierSave.mutate(draft)}/>}
    {detailId&&(detail.isLoading?<RemoteModalSkeleton className="purchase-detail-modal" label="Cargando orden de compra" rows={6} close={()=>setDetailId(null)}/>:detail.isError?<PurchaseDetailError message={detail.error.message} close={()=>setDetailId(null)}/>:detail.data&&<PurchaseDetail order={detail.data} canManage={canManage} canReceive={canReceive} busy={transition.isPending} currency={formatMoney(Number(detail.data.total),settings,location?.country)} country={location?.country} timezone={location?.timezone} close={()=>setDetailId(null)} edit={()=>{setDetailId(null);void editOrder(detail.data.id)}} changeStatus={next=>transition.mutate({id:detail.data.id,next})} receive={()=>{setReceiptOrder(detail.data);setDetailId(null)}} cancel={()=>setCancelTarget(detail.data)}/>)}
    {receiptOrder&&<PurchaseReceiptDialog order={receiptOrder} busy={receive.isPending} close={()=>setReceiptOrder(null)} save={draft=>receive.mutate(draft)}/>}
    <ConfirmDialog open={Boolean(supplierTarget)} title={supplierTarget?.active?"Desactivar proveedor":"Activar proveedor"} description={supplierTarget?.active?`“${supplierTarget?.name??""}” dejará de estar disponible para nuevas órdenes. Su historial se conservará.`:`“${supplierTarget?.name??""}” volverá a estar disponible para nuevas órdenes.`} tone={supplierTarget?.active?"danger":"success"} confirmLabel={supplierTarget?.active?"Desactivar":"Activar"} pending={supplierStatus.isPending} onCancel={()=>setSupplierTarget(null)} onConfirm={()=>supplierTarget&&supplierStatus.mutate(supplierTarget)}/>
    <ConfirmDialog open={Boolean(cancelTarget)} title="Cancelar orden de compra" description={`“${cancelTarget?.number??""}” quedará cancelada y ya no podrá recibirse. El historial se conservará.`} tone="danger" confirmLabel="Cancelar orden" pending={transition.isPending} onCancel={()=>setCancelTarget(null)} onConfirm={()=>cancelTarget&&transition.mutate({id:cancelTarget.id,next:"cancelled"})}/>
  </>;
}

function PurchaseOrderDialog({initial,suppliers,inventory,currencySymbol,busy,close,save}:{initial:PurchaseOrderDraft;suppliers:Supplier[];inventory:PurchaseInventoryOption[];currencySymbol:string;busy:boolean;close:()=>void;save:(draft:PurchaseOrderDraft)=>void}){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const[createdItems,setCreatedItems]=useState<PurchaseInventoryOption[]>([]);
  const[itemSearch,setItemSearch]=useState<Record<number,string>>({});
  const[newItemLine,setNewItemLine]=useState<number|null>(null);
  const{control,register,handleSubmit,watch,setValue,formState:{errors,isSubmitted}}=useForm<PurchaseOrderDraft>({defaultValues:initial,resolver:purchaseOrderResolver,mode:"onSubmit",reValidateMode:"onChange"});
  const{fields,append,remove}=useFieldArray({control,name:"items"});
  const categories=useQuery({queryKey:["purchase-item-categories"],queryFn:listPurchaseItemCategories,enabled:newItemLine!==null,staleTime:30000});
  const createItem=useMutation({
    mutationFn:({draft}:{index:number;draft:Parameters<typeof createPurchaseInventoryItem>[0]})=>createPurchaseInventoryItem(draft),
    onSuccess:(item,variables)=>{
      setCreatedItems(current=>current.some(existing=>existing.id===item.id)?current:[...current,item]);
      chooseItem(variables.index,item.id,item);
      setNewItemLine(null);
      void qc.invalidateQueries({queryKey:["purchase-inventory"]});
      void qc.invalidateQueries({queryKey:["inventory"]});
      void qc.invalidateQueries({queryKey:["products"]});
      notify({tone:"success",title:"Artículo creado",message:item.name+" quedó con stock 0 y seleccionado en la orden."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo crear el artículo",message:error.message}),
  });
  const lines=watch("items");
  const catalog=[...inventory,...createdItems.filter(item=>!inventory.some(existing=>existing.id===item.id))];
  const total=lines.reduce((sum,line)=>sum+(Number(line.quantity)||0)*(Number(line.unitCost)||0),0);

  function chooseItem(index:number,id:string,forced?:PurchaseInventoryOption){
    const option=forced??catalog.find(item=>item.id===id);
    const presentation=option?.presentations.find(item=>item.presentationType==="unit")??option?.presentations[0];
    setValue(\`items.\${index}.inventoryItemId\`,id,{shouldValidate:isSubmitted});
    setValue(\`items.\${index}.presentationId\`,presentation?.id??"",{shouldValidate:isSubmitted});
    setItemSearch(current=>({...current,[index]:""}));
  }

  return <>
    <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-order-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-order-title" aria-busy={busy}><div className="modal-accent"/>
      <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>{initial.id?"EDITAR ORDEN":"NUEVA ORDEN"}</small><h2 id="purchase-order-title">Orden de compra</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
      <form onSubmit={handleSubmit(save)} noValidate><div className="purchase-form-body">
        <section className="purchase-form-section"><div className="purchase-section-title"><span><Icon name="truck" size={17}/></span><div><b>Proveedor y entrega</b><small>Define quién abastece la orden y cuándo esperas recibirla.</small></div></div><div className="form-grid">
          <label>Proveedor<Select autoFocus {...register("supplierId")} aria-invalid={Boolean(errors.supplierId)}><option value="">{suppliers.length?"Selecciona un proveedor":"No hay proveedores activos"}</option>{suppliers.map(supplier=><option value={supplier.id} key={supplier.id}>{supplier.name}{supplier.taxId?" · "+supplier.taxId:""}</option>)}</Select>{errors.supplierId?.message&&<small className="wizard-field-error">{errors.supplierId.message}</small>}</label>
          <label>Entrega esperada<Input type="date" {...register("expectedAt")}/></label><div/>
          <label className="span-2">Notas<Textarea rows={2} maxLength={500} {...register("notes")} placeholder="Condiciones, referencia o indicaciones para la compra"/>{errors.notes?.message&&<small className="wizard-field-error">{errors.notes.message}</small>}</label>
        </div></section>
        <section className="purchase-form-section"><div className="purchase-section-title purchase-lines-title"><span><Icon name="stock" size={17}/></span><div><b>Artículos de la orden</b><small>Busca un artículo existente o créalo aquí si todavía no existe.</small></div><Button kind="secondary" icon="plus" onClick={()=>append({inventoryItemId:"",presentationId:"",quantity:"1",unitCost:""})}>Agregar línea</Button></div>
          <div className="purchase-lines">{fields.map((field,index)=>{
            const value=lines[index];
            const selected=catalog.find(item=>item.id===value?.inventoryItemId);
            const search=(itemSearch[index]??"").trim().toLowerCase();
            let options=search?catalog.filter(item=>item.name.toLowerCase().includes(search)):catalog;
            if(selected&&!options.some(item=>item.id===selected.id))options=[selected,...options];
            return <article className="purchase-line" key={field.id}>
              <div className="purchase-line-number">{index+1}</div>
              <label className="purchase-item-picker">Artículo
                <Input value={itemSearch[index]??""} onChange={event=>setItemSearch(current=>({...current,[index]:event.target.value}))} placeholder="Buscar artículo existente..."/>
                <Select value={value?.inventoryItemId??""} onChange={event=>chooseItem(index,event.target.value)} aria-invalid={Boolean(errors.items?.[index]?.inventoryItemId)}>
                  <option value="">{options.length?"Selecciona...":"Sin coincidencias"}</option>
                  {options.map(item=><option value={item.id} key={item.id}>{item.name}{item.kind==="ingredient"?" · Insumo":""}</option>)}
                </Select>
                <button type="button" className="purchase-create-item" onClick={()=>setNewItemLine(index)}><Icon name="plus" size={13}/>{search&&options.length===0?"Crear nuevo artículo":"Crear nuevo artículo"}</button>
                {errors.items?.[index]?.inventoryItemId?.message&&<small className="wizard-field-error">{errors.items[index]?.inventoryItemId?.message}</small>}
              </label>
              <label>Presentación<Select {...register(\`items.\${index}.presentationId\`)} disabled={!selected} aria-invalid={Boolean(errors.items?.[index]?.presentationId)}><option value="">Selecciona...</option>{(selected?.presentations??[]).map(presentation=><option value={presentation.id} key={presentation.id}>{presentationName(presentation.presentationType,presentation.unitsPerPresentation,selected?.unit)}</option>)}</Select>{errors.items?.[index]?.presentationId?.message&&<small className="wizard-field-error">{errors.items[index]?.presentationId?.message}</small>}</label>
              <label>Cantidad<Input type="number" min="0.001" step="0.001" inputMode="decimal" {...register(\`items.\${index}.quantity\`)} aria-invalid={Boolean(errors.items?.[index]?.quantity)}/>{errors.items?.[index]?.quantity?.message&&<small className="wizard-field-error">{errors.items[index]?.quantity?.message}</small>}</label>
              <label>Costo unitario<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" {...register(\`items.\${index}.unitCost\`)} placeholder="0.00" aria-invalid={Boolean(errors.items?.[index]?.unitCost)}/></div>{errors.items?.[index]?.unitCost?.message&&<small className="wizard-field-error">{errors.items[index]?.unitCost?.message}</small>}</label>
              <div className="purchase-line-total"><small>SUBTOTAL</small><b>{currencySymbol} {formatRegionalNumber((Number(value?.quantity)||0)*(Number(value?.unitCost)||0),undefined,{minimumFractionDigits:2,maximumFractionDigits:2})}</b></div>
              <RowActionButton action="remove" label="Quitar línea" disabled={fields.length===1} onClick={()=>remove(index)}/>
            </article>;
          })}</div>
          {typeof errors.items?.message==="string"&&<div className="purchase-validation" role="alert"><Icon name="alert" size={15}/>{errors.items.message}</div>}
        </section>
        <div className="purchase-total"><span><small>TOTAL ESTIMADO</small><b>{fields.length} {fields.length===1?"línea":"líneas"}</b></span><strong>{currencySymbol} {formatRegionalNumber(total,undefined,{minimumFractionDigits:2,maximumFractionDigits:2})}</strong></div>
      </div><footer><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||!suppliers.length}>{busy?"Guardando…":"Guardar"}</Button></footer></form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </section></div>
    {newItemLine!==null&&(categories.isLoading?<RemoteModalSkeleton className="purchase-item-modal" label="Cargando catálogo para el artículo" rows={6} close={()=>setNewItemLine(null)}/>:<PurchaseItemDialog categories={categories.data??[]} categoryError={categories.isError?categories.error.message:null} currencySymbol={currencySymbol} busy={createItem.isPending} close={()=>setNewItemLine(null)} save={draft=>createItem.mutate({index:newItemLine,draft})}/>)}
  </>;
}

function SupplierDialog({initial,busy,close,save}:{initial:SupplierDraft;busy:boolean;close:()=>void;save:(draft:SupplierDraft)=>void}){
  const{register,handleSubmit,formState:{errors}}=useForm<SupplierDraft>({defaultValues:initial,resolver:supplierResolver,mode:"onSubmit",reValidateMode:"onChange"});
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal supplier-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="supplier-title" aria-busy={busy}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="truck" size={18}/></span><div><small>{initial.id?"EDITAR PROVEEDOR":"NUEVO PROVEEDOR"}</small><h2 id="supplier-title">Datos del proveedor</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header><form onSubmit={handleSubmit(save)} noValidate><div className="supplier-form-body form-grid">
    <label className="span-2">Nombre o razón social<Input autoFocus {...register("name")} aria-invalid={Boolean(errors.name)} placeholder="Ej. Distribuidora Andina"/>{errors.name?.message&&<small className="wizard-field-error">{errors.name.message}</small>}</label>
    <label>RUC<Input inputMode="numeric" maxLength={11} {...register("taxId")} aria-invalid={Boolean(errors.taxId)} placeholder="20123456789"/>{errors.taxId?.message&&<small className="wizard-field-error">{errors.taxId.message}</small>}</label>
    <label>Teléfono<Input {...register("phone")} placeholder="Ej. 987 654 321"/></label>
    <label className="span-2">Correo electrónico<Input type="email" {...register("email")} aria-invalid={Boolean(errors.email)} placeholder="compras@proveedor.com"/>{errors.email?.message&&<small className="wizard-field-error">{errors.email.message}</small>}</label>
  </div><footer><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button></footer></form>{busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}</section></div>;
}

function PurchaseDetail({order,canManage,canReceive,busy,currency,country,timezone,close,edit,changeStatus,receive,cancel}:{order:PurchaseOrder;canManage:boolean;canReceive:boolean;busy:boolean;currency:string;country?:string|null;timezone?:string|null;close:()=>void;edit:()=>void;changeStatus:(next:Exclude<PurchaseStatus,"received">)=>void;receive:()=>void;cancel:()=>void}){
  const meta=statusMeta[order.status];
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-detail-title" aria-busy={busy}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>ORDEN DE COMPRA</small><h2 id="purchase-detail-title">{order.number}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header><div className="purchase-detail-body">
    <section className="purchase-detail-summary"><div><small>PROVEEDOR</small><b>{order.supplierName}</b></div><div><small>TOTAL</small><b>{currency}</b></div><div><small>ARTÍCULOS</small><b>{order.itemCount}</b></div><Status tone={meta.tone}>{meta.label}</Status></section>
    <section className="purchase-detail-meta"><div><small>CREADA</small><b>{formatRegionalDateTime(order.createdAt,{country,timeZone:timezone},{dateStyle:"medium",timeStyle:"short"})}</b></div><div><small>ENTREGA ESPERADA</small><b>{order.expectedAt?formatRegionalCalendarDate(order.expectedAt,country,{dateStyle:"medium"}):"Sin fecha"}</b></div><div><small>NOTAS</small><b>{order.notes||"Sin notas"}</b></div></section>
    <section className="purchase-detail-lines"><header><div><small>DETALLE</small><h3>Artículos solicitados</h3></div></header><div className="table-wrap hover-scroll"><table><thead><tr><th>ARTÍCULO</th><th>PRESENTACIÓN</th><th>CANTIDAD</th><th>COSTO</th><th>SUBTOTAL</th></tr></thead><tbody>{order.items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}><td><b>{item.itemName}</b><small>{item.sku||item.unit}</small></td><td>{presentationName(item.presentationType,item.unitsPerPresentation,item.unit)}</td><td>{formatRegionalNumber(Number(item.quantity),country,{maximumFractionDigits:3})}</td><td>{formatRegionalNumber(Number(item.unitCost),country,{minimumFractionDigits:2,maximumFractionDigits:4})}</td><td><b>{formatRegionalNumber(Number(item.lineTotal),country,{minimumFractionDigits:2,maximumFractionDigits:2})}</b></td></tr>)}</tbody></table></div></section>
    {(canManage||canReceive)&&order.status!=="received"&&order.status!=="cancelled"&&<section className="purchase-detail-actions"><div><small>SIGUIENTE PASO</small><b>{nextStepLabel(order.status)}</b></div><div>{canManage&&order.status==="draft"&&<><Button kind="secondary" icon="edit" onClick={edit} disabled={busy}>Editar</Button><Button icon="chevron" onClick={()=>changeStatus("pending_approval")} disabled={busy}>Enviar a aprobación</Button></>}{canManage&&order.status==="pending_approval"&&<><Button kind="ghost" icon="chevronLeft" onClick={()=>changeStatus("draft")} disabled={busy}>Volver a borrador</Button><Button icon="check" onClick={()=>changeStatus("approved")} disabled={busy}>Aprobar</Button></>}{canReceive&&order.status==="approved"&&<Button kind="success" icon="stock" onClick={receive} disabled={busy}>Recibir compra</Button>}{canManage&&<Button kind="danger" icon="close" onClick={cancel} disabled={busy}>Cancelar orden</Button>}</div></section>}
  </div>{busy&&<div className="modal-busy" role="status"><i/><span>Procesando…</span></div>}</section></div>;
}

function PurchaseDetailError({message,close}:{message:string;close:()=>void}){return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-detail-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="alert"/></span><div><small>ORDEN DE COMPRA</small><h2>No pudimos cargar el detalle</h2></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><PurchaseState icon="alert" title="Detalle no disponible" text={message}/></section></div>}

function PurchaseEditorError({message,close,retry}:{message:string;close:()=>void;retry:()=>void}){return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-order-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="alert"/></span><div><small>ORDEN DE COMPRA</small><h2>No pudimos preparar el formulario</h2></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><PurchaseState icon="alert" title="Catálogos no disponibles" text={message} action={retry}/></section></div>}

function presentationName(type:string,factor:string,unit?:string){if(type==="unit")return `Unidad base · ${unit??"und"}`;return `${type==="box"?"Caja":"Paquete"} x ${formatRegionalNumber(Number(factor),undefined,{maximumFractionDigits:3})} ${unit??"und"}`}
function nextStepLabel(status:PurchaseStatus){if(status==="draft")return"Completa la orden y envíala para aprobación.";if(status==="pending_approval")return"Revisa el total y aprueba antes de recibir.";if(status==="approved")return"Confirma la recepción para ingresar existencias.";return"Sin acciones pendientes."}
function formatMoney(amount:number,settings:{currencySymbol:string;currencyPosition:"before"|"after";currencyDecimals:number},country?:string|null){const value=formatRegionalNumber(amount,country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});return settings.currencyPosition==="before"?`${settings.currencySymbol} ${value}`:`${value} ${settings.currencySymbol}`}

function PurchaseState({icon,title,text,action}:{icon:"alert"|"receipt"|"truck";title:string;text:string;action?:()=>void}){return <div className="catalog-state purchases-state"><span><Icon name={icon}/></span><b>{title}</b><p>{text}</p>{action&&<Button kind="secondary" icon={icon==="alert"?"refresh":"plus"} onClick={action}>{icon==="alert"?"Reintentar":icon==="truck"?"Nuevo proveedor":"Nueva orden"}</Button>}</div>}
function PurchaseTableSkeleton(){return <div className="purchases-loading" aria-label="Cargando compras">{Array.from({length:6},(_,row)=><i key={row}/>)}</div>}
function SupplierTableSkeleton(){return <div className="purchases-loading" aria-label="Cargando proveedores">{Array.from({length:6},(_,row)=><i key={row}/>)}</div>}
