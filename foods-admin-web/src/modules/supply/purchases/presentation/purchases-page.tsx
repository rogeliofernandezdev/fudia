"use client";
import dynamic from "next/dynamic";
import "./purchases.css";
import {useState} from "react";
import {useFieldArray,useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status,Textarea} from "@/design-system";
import {useFeedback,useSession,useSettings} from "@/providers";
import {formatRegionalCalendarDate,formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {purchaseOrderResolver,supplierResolver} from "../domain/purchase-schema";
const PurchaseItemDialog=dynamic(()=>import("./purchase-item-dialog").then(module=>module.PurchaseItemDialog),{ssr:false});
const PurchaseReceiptDialog=dynamic(()=>import("./purchase-receipt-dialog").then(module=>module.PurchaseReceiptDialog),{ssr:false});
const PurchaseReturnDialog=dynamic(()=>import("./purchase-return-dialog").then(module=>module.PurchaseReturnDialog),{ssr:false});
import type {PurchaseInventoryOption,PurchaseOrder,PurchaseOrderDraft,PurchaseOrderSummary,PurchaseReceiptDetail,PurchaseStatus,PurchaseTab,Supplier,SupplierDraft} from "../domain/types";
import {approvePurchaseOrder,createPurchaseInventoryItem,createPurchaseReturn,getPurchaseOrder,getPurchaseReceipt,listPurchaseInventory,listPurchaseItemCategories,listPurchaseOrders,listPurchaseReceipts,listSuppliers,receivePurchaseOrder,savePurchaseOrder,saveSupplier,setPurchaseOrderStatus,setSupplierActive} from "../infrastructure/purchases-api";

const statusMeta:Record<PurchaseStatus,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={
  draft:{label:"Borrador",tone:"gray"},
  pending_approval:{label:"Por aprobar",tone:"orange"},
  approved:{label:"Aprobada",tone:"blue"},
  partially_received:{label:"Recepción parcial",tone:"orange"},
  received:{label:"Recibida",tone:"green"},
  cancelled:{label:"Cancelada",tone:"gray"},
};
const emptyOrder:PurchaseOrderDraft={id:"",supplierId:"",expectedAt:"",notes:"",items:[]};
const emptySupplier:SupplierDraft={id:"",taxId:"",name:"",email:"",phone:""};

export function PurchasesPage(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const settings=useSettings();
  const canManage=can("purchases.manage");
  const canApprove=can("purchases.approve");
  const canReceive=can("purchases.receive");
  const[tab,setTab]=useState<PurchaseTab>("orders");
  const[q,setQ]=useState("");
  const debouncedQ=useDebouncedValue(q);
  const[status,setStatus]=useState("");
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[orderDraft,setOrderDraft]=useState<PurchaseOrderDraft|null>(null);
  const[orderDraftLoading,setOrderDraftLoading]=useState(false);
  const[detailId,setDetailId]=useState<string|null>(null);
  const[detailMode,setDetailMode]=useState<"view"|"review">("view");
  const[supplierDraft,setSupplierDraft]=useState<SupplierDraft|null>(null);
  const[supplierTarget,setSupplierTarget]=useState<Supplier|null>(null);
  const[cancelTarget,setCancelTarget]=useState<PurchaseOrderSummary|PurchaseOrder|null>(null);
  const[receiptOrder,setReceiptOrder]=useState<PurchaseOrder|null>(null);
  const[receiptLoadingId,setReceiptLoadingId]=useState<string|null>(null);
  const[receiptView,setReceiptView]=useState<"pending"|"history">("pending");
  const[receiptDetail,setReceiptDetail]=useState<PurchaseReceiptDetail|null>(null);
  const[receiptDetailLoading,setReceiptDetailLoading]=useState("");
  const[returnReceipt,setReturnReceipt]=useState<PurchaseReceiptDetail|null>(null);

  const orders=useQuery({
    queryKey:["purchase-orders",debouncedQ,status,page,size],
    queryFn:()=>listPurchaseOrders({q:debouncedQ,status,page,pageSize:size}),
    enabled:tab==="orders",
  });
  const receipts=useQuery({
    queryKey:["purchase-orders","receipts",debouncedQ,status,page,size],
    queryFn:()=>listPurchaseOrders({q:debouncedQ,status:status||"receivable",page,pageSize:size}),
    enabled:tab==="receipts",
  });
  const receivableSummary=useQuery({
    queryKey:["purchase-orders","receivable-summary"],
    queryFn:()=>listPurchaseOrders({q:"",status:"receivable",page:1,pageSize:1}),
  });
  const receiptHistory=useQuery({
    queryKey:["purchase-receipts",debouncedQ,page,size],
    queryFn:()=>listPurchaseReceipts({q:debouncedQ,page,pageSize:size}),
    enabled:tab==="receipts"&&receiptView==="history",
  });
  const suppliers=useQuery({
    queryKey:["suppliers",debouncedQ,status,page,size],
    queryFn:()=>listSuppliers({q:debouncedQ,status,page,pageSize:size}),
    enabled:tab==="suppliers",
  });
  const activeSuppliers=useQuery({
    queryKey:["suppliers","active","purchase-editor"],
    queryFn:()=>listSuppliers({status:"active",page:1,pageSize:100}),
    enabled:Boolean(orderDraft),
    staleTime:30000,
  });
  const inventory=useQuery({
    queryKey:["purchase-inventory"],
    queryFn:()=>listPurchaseInventory(),
    enabled:Boolean(orderDraft),
    staleTime:30000,
  });
  const detail=useQuery({
    queryKey:["purchase-order",detailId],
    queryFn:()=>getPurchaseOrder(detailId!),
    enabled:Boolean(detailId),
  });

  const saveOrder=useMutation({
    mutationFn:savePurchaseOrder,
    onSuccess:result=>{
      setOrderDraft(null);
      void qc.invalidateQueries({queryKey:["purchase-orders"]});
      void qc.invalidateQueries({queryKey:["dashboard"]});
      notify({tone:"success",title:"Orden guardada",message:result.number+" quedó en borrador y lista para revisión."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo guardar la orden",message:error.message}),
  });

  const transition=useMutation({
    mutationFn:({id,next}:{id:string;next:"draft"|"pending_approval"|"cancelled"})=>setPurchaseOrderStatus(id,next),
    onSuccess:(_,variables)=>{
      void qc.invalidateQueries({queryKey:["purchase-orders"]});
      void qc.invalidateQueries({queryKey:["purchase-order"]});
      void qc.invalidateQueries({queryKey:["dashboard"]});
      if(variables.next==="cancelled")setCancelTarget(null);
      notify({tone:"success",title:"Estado actualizado",message:"La orden de compra quedó actualizada."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo actualizar la orden",message:error.message}),
  });
  const approve=useMutation({
    mutationFn:approvePurchaseOrder,
    onSuccess:()=>{void qc.invalidateQueries({queryKey:["purchase-orders"]});void qc.invalidateQueries({queryKey:["purchase-order"]});void qc.invalidateQueries({queryKey:["dashboard"]});notify({tone:"success",title:"Orden aprobada",message:"La orden ya está disponible para recepción."})},
    onError:error=>notify({tone:"danger",title:"No se pudo aprobar",message:error.message}),
  });

  const receive=useMutation({
    mutationFn:receivePurchaseOrder,
    onSuccess:result=>{
      setReceiptOrder(null);
      void qc.invalidateQueries({queryKey:["purchase-orders"]});
      void qc.invalidateQueries({queryKey:["purchase-order"]});
      void qc.invalidateQueries({queryKey:["purchase-inventory"]});
      void qc.invalidateQueries({queryKey:["inventory"]});
      void qc.invalidateQueries({queryKey:["inventory-products"]});
      void qc.invalidateQueries({queryKey:["inventory-movements"]});
      void qc.invalidateQueries({queryKey:["purchase-receipts"]});
      void qc.invalidateQueries({queryKey:["dashboard"]});
      notify({
        tone:"success",
        title:result.status==="received"?"Recepción completada":"Recepción parcial registrada",
        message:result.code+" quedó vinculada a "+result.number+" y actualizó Inventario y Kárdex.",
      });
    },
    onError:error=>notify({tone:"danger",title:"No se pudo registrar la recepción",message:error.message}),
  });

  const returnPurchase=useMutation({
    mutationFn:createPurchaseReturn,
    onSuccess:result=>{setReturnReceipt(null);setReceiptDetail(null);void qc.invalidateQueries({queryKey:["purchase-receipts"]});void qc.invalidateQueries({queryKey:["purchase-orders"]});void qc.invalidateQueries({queryKey:["inventory"]});void qc.invalidateQueries({queryKey:["inventory-products"]});void qc.invalidateQueries({queryKey:["inventory-movements"]});notify({tone:"success",title:result.kind==="receipt_correction"?"Recepción corregida":"Devolución registrada",message:result.code+" actualizó Inventario y Kárdex."})},
    onError:error=>notify({tone:"danger",title:"No se pudo procesar",message:error.message}),
  });

  const supplierSave=useMutation({
    mutationFn:saveSupplier,
    onSuccess:()=>{
      setSupplierDraft(null);
      void qc.invalidateQueries({queryKey:["suppliers"]});
      notify({tone:"success",title:"Proveedor guardado",message:"El directorio de proveedores quedó actualizado."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo guardar el proveedor",message:error.message}),
  });

  const supplierStatus=useMutation({
    mutationFn:(supplier:Supplier)=>setSupplierActive(supplier.id,!supplier.active),
    onSuccess:()=>{
      setSupplierTarget(null);
      void qc.invalidateQueries({queryKey:["suppliers"]});
      notify({tone:"success",title:"Estado actualizado",message:"El proveedor conserva su historial de compras."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo actualizar el proveedor",message:error.message}),
  });

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
        items:order.items.map(item=>({
          inventoryItemId:item.inventoryItemId,
          presentationId:item.presentationId,
          quantity:item.quantity,
          unitCost:item.unitCost,
        })),
      });
    }catch(error){
      setOrderDraft(null);
      notify({tone:"danger",title:"No se pudo cargar la orden",message:(error as Error).message});
    }finally{
      setOrderDraftLoading(false);
    }
  }

  function openDetail(id:string,mode:"view"|"review"){
    setDetailMode(mode);
    setDetailId(id);
  }

  async function openReceipt(id:string){
    setReceiptLoadingId(id);
    try{
      const order=await getPurchaseOrder(id);
      setReceiptOrder(order);
    }catch(error){
      notify({tone:"danger",title:"No se pudo preparar la recepción",message:(error as Error).message});
    }finally{
      setReceiptLoadingId(null);
    }
  }
  async function openReceiptDetail(id:string){
    setReceiptDetailLoading(id);
    try{setReceiptDetail(await getPurchaseReceipt(id))}
    catch(error){notify({tone:"danger",title:"No se pudo cargar la recepción",message:(error as Error).message})}
    finally{setReceiptDetailLoading("")}
  }

  function changeTab(next:PurchaseTab){
    if(next==="receipts")void qc.invalidateQueries({queryKey:["purchase-orders","receipts"]});
    setTab(next);
    setQ("");
    setStatus("");
    setPage(1);
  }

  const orderItems=orders.data?.items??[];
  const receiptItems=receipts.data?.items??[];
  const receiptHistoryItems=receiptHistory.data?.items??[];
  const supplierItems=suppliers.data?.items??[];
  const headerAction=tab==="orders"
    ?canManage?<Button icon="plus" onClick={()=>setOrderDraft({...emptyOrder})}>Nueva orden</Button>:undefined
    :tab==="suppliers"&&canManage
      ?<Button icon="plus" onClick={()=>setSupplierDraft({...emptySupplier})}>Nuevo proveedor</Button>
      :undefined;

  return <>
    <PageHeader
      eyebrow="ABASTECIMIENTO"
      title="Compras"
      description="Crea órdenes de compra, recibe mercadería y administra proveedores desde flujos separados."
      action={headerAction}
    />

    <section className="panel standardized-management purchases-panel">
      <div className="purchases-tabs" role="tablist" aria-label="Compras">
        <button type="button" role="tab" aria-selected={tab==="orders"} className={tab==="orders"?"active":""} onClick={()=>changeTab("orders")}>
          <Icon name="receipt" size={17}/><span>Órdenes de compra</span>{orders.data&&<b>{orders.data.total}</b>}
        </button>
        <button type="button" role="tab" aria-selected={tab==="receipts"} className={tab==="receipts"?"active":""} onClick={()=>changeTab("receipts")}>
          <Icon name="stock" size={17}/><span>Recepciones</span>{receivableSummary.data&&<b>{receivableSummary.data.total}</b>}
        </button>
        <button type="button" role="tab" aria-selected={tab==="suppliers"} className={tab==="suppliers"?"active":""} onClick={()=>changeTab("suppliers")}>
          <Icon name="truck" size={17}/><span>Proveedores</span>{suppliers.data&&<b>{suppliers.data.total}</b>}
        </button>
      </div>

      <div className="purchases-toolbar">
        <label className="purchases-search">
          <Icon name="search" size={18}/>
          <Input
            value={q}
            onChange={event=>{setQ(event.target.value);setPage(1)}}
            placeholder={tab==="suppliers"?"Buscar proveedor o RUC...":tab==="receipts"?(receiptView==="history"?"Buscar REC, OC o proveedor...":"Buscar orden pendiente de recepción..."):"Buscar por orden o proveedor..."}
          />
        </label>

        {tab==="orders"?<Select aria-label="Filtrar órdenes por estado" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
          <option value="">Todos los estados</option>
          {Object.entries(statusMeta).map(([value,meta])=><option value={value} key={value}>{meta.label}</option>)}
        </Select>:tab==="receipts"?(receiptView==="pending"?<Select aria-label="Filtrar recepciones" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
          <option value="">Todas pendientes</option>
          <option value="approved">Sin recibir</option>
          <option value="partially_received">Recepción parcial</option>
        </Select>:<span/>):<Select aria-label="Filtrar proveedores por estado" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
          <option value="">Todos los estados</option>
          <option value="active">Activos</option>
          <option value="inactive">Inactivos</option>
        </Select>}

        <p>
          <Icon name="store" size={15}/>
          {tab==="orders"
            ?"La orden registra lo solicitado y no modifica el stock."
            :tab==="receipts"
              ?(receiptView==="pending"?"Órdenes aprobadas con cantidades pendientes.":"Historial documental de recepciones, correcciones y devoluciones.")
              :"Proveedores compartidos por la empresa para abastecer cualquiera de sus locales."}
        </p>
      </div>

      {tab==="orders"?<>
        {orders.isLoading?<PurchaseTableSkeleton/>
        :orders.isError?<PurchaseState icon="alert" title="No pudimos cargar las órdenes" text={orders.error.message} action={()=>orders.refetch()}/>
        :!orderItems.length?<PurchaseState
          icon="receipt"
          title={q||status?"Sin coincidencias":"Aún no hay órdenes de compra"}
          text={q||status?"Ajusta la búsqueda o los filtros.":canManage?"Crea una orden y agrega los artículos que necesitas abastecer.":"No hay órdenes registradas para este local."}
          action={canManage&&!q&&!status?()=>setOrderDraft({...emptyOrder}):undefined}
        />:<>
          <div className="table-wrap hover-scroll purchases-table-wrap"><table>
            <thead><tr><th>ORDEN</th><th>PROVEEDOR</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th><th>ACCIONES</th></tr></thead>
            <tbody>{orderItems.map((order,index)=>{const meta=statusMeta[order.status];return <tr className={index%2?"alternate":""} key={order.id}>
              <td><span className={"row-icon r"+index%3}><Icon name="receipt" size={18}/></span><b>{order.number}</b><small>{order.itemCount} {order.itemCount===1?"artículo":"artículos"}</small></td>
              <td><b>{order.supplierName}</b>{order.expectedAt&&<small>Entrega esperada: {formatRegionalCalendarDate(order.expectedAt,location?.country,{dateStyle:"medium"})}</small>}</td>
              <td>{formatRegionalDateTime(order.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium"})}</td>
              <td><b className="purchase-money">{formatMoney(Number(order.total),settings,location?.country)}</b></td>
              <td><Status tone={meta.tone}>{meta.label}</Status></td>
              <td><div className="table-actions">
                <RowActionButton action="view" onClick={()=>openDetail(order.id,"view")}/>
                {(canManage||canApprove||canReceive)&&["draft","pending_approval","approved","partially_received"].includes(order.status)&&<RowActionButton action="review" onClick={()=>openDetail(order.id,"review")}/>} 
              </div></td>
            </tr>})}</tbody>
          </table></div>

          <div className="management-cards purchases-cards">{orderItems.map(order=>{const meta=statusMeta[order.status];return <article key={order.id}>
            <header><span className="row-icon r0"><Icon name="receipt"/></span><div><b>{order.number}</b><small>{order.supplierName}</small></div><Status tone={meta.tone}>{meta.label}</Status></header>
            <dl><div><dt>Artículos</dt><dd>{order.itemCount}</dd></div><div><dt>Total</dt><dd>{formatMoney(Number(order.total),settings,location?.country)}</dd></div><div><dt>Creada</dt><dd>{formatRegionalDateTime(order.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"short"})}</dd></div></dl>
            <footer><RowActionButton action="view" onClick={()=>openDetail(order.id,"view")}/>{(canManage||canApprove||canReceive)&&["draft","pending_approval","approved","partially_received"].includes(order.status)&&<RowActionButton action="review" onClick={()=>openDetail(order.id,"review")}/>} </footer>
          </article>})}</div>
        </>}
        {!orders.isLoading&&!orders.isError&&<Pagination page={page} size={size} total={orders.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
      </>:tab==="receipts"?<>
        <div className="purchases-tabs" style={{marginBottom:12}}>
          <button type="button" className={receiptView==="pending"?"active":""} onClick={()=>{setReceiptView("pending");setPage(1);setStatus("")}}><Icon name="stock" size={15}/>Pendientes<b>{receivableSummary.data?.total??0}</b></button>
          <button type="button" className={receiptView==="history"?"active":""} onClick={()=>{setReceiptView("history");setPage(1);setStatus("")}}><Icon name="receipt" size={15}/>Historial</button>
        </div>
        {receiptView==="history"?<>
          {receiptHistory.isLoading?<PurchaseTableSkeleton/>
          :receiptHistory.isError?<PurchaseState icon="alert" title="No pudimos cargar el historial" text={receiptHistory.error.message} action={()=>receiptHistory.refetch()}/>
          :!receiptHistoryItems.length?<div className="purchase-receipts-empty"><span><Icon name="receipt" size={22}/></span><b>Sin recepciones registradas</b><p>Las recepciones confirmadas aparecerán aquí.</p></div>
          :<div className="table-wrap hover-scroll purchases-table-wrap"><table><thead><tr><th>RECEPCIÓN</th><th>OC</th><th>PROVEEDOR</th><th>FECHA</th><th>ARTÍCULOS</th><th>USUARIO</th><th>ACCIÓN</th></tr></thead><tbody>{receiptHistoryItems.map((r,index)=><tr className={index%2?"alternate":""} key={r.id}><td><b>{r.code}</b></td><td>{r.number}</td><td>{r.supplierName}</td><td>{formatRegionalDateTime(r.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"})}</td><td>{r.itemCount}</td><td>{r.createdByName}</td><td><Button kind="secondary" disabled={receiptDetailLoading===r.id} onClick={()=>void openReceiptDetail(r.id)}>{receiptDetailLoading===r.id?"Cargando…":"Ver recepción"}</Button></td></tr>)}</tbody></table></div>}
          {!receiptHistory.isLoading&&!receiptHistory.isError&&<Pagination page={page} size={size} total={receiptHistory.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
        </>:<>
        {receipts.isLoading?<PurchaseTableSkeleton/>
        :receipts.isError?<PurchaseState icon="alert" title="No pudimos cargar las recepciones pendientes" text={receipts.error.message} action={()=>receipts.refetch()}/>
        :!receiptItems.length?<div className="purchase-receipts-empty">
          <span><Icon name="check" size={22}/></span>
          <b>{q||status?"Sin coincidencias":"No hay mercadería pendiente de recibir"}</b>
          <p>{q||status?"Ajusta la búsqueda o el filtro.":"Solo aparecen órdenes aprobadas con cantidades pendientes."}</p>
        </div>:<>
          <div className="table-wrap hover-scroll purchases-table-wrap purchase-receipts-table"><table>
            <thead><tr><th>ORDEN</th><th>PROVEEDOR</th><th>ENTREGA ESPERADA</th><th>ARTÍCULOS</th><th>ESTADO</th><th>ACCIÓN</th></tr></thead>
            <tbody>{receiptItems.map((order,index)=>{const meta=statusMeta[order.status];const loading=receiptLoadingId===order.id;return <tr className={index%2?"alternate":""} key={order.id}>
              <td><span className={"row-icon r"+index%3}><Icon name="stock" size={18}/></span><b>{order.number}</b><small>Aprobada para recepción</small></td>
              <td><b>{order.supplierName}</b></td>
              <td>{order.expectedAt?formatRegionalCalendarDate(order.expectedAt,location?.country,{dateStyle:"medium"}):"Sin fecha"}</td>
              <td>{order.itemCount}</td>
              <td><Status tone={meta.tone}>{meta.label}</Status></td>
              <td><div className="purchase-receipt-actions">
                <RowActionButton action="view" onClick={()=>openDetail(order.id,"view")}/>
                {canReceive&&<Button kind="success" icon="stock" disabled={loading} onClick={()=>void openReceipt(order.id)}>{loading?"Cargando…":order.status==="partially_received"?"Continuar":"Recibir"}</Button>}
              </div></td>
            </tr>})}</tbody>
          </table></div>

          <div className="management-cards purchases-cards purchase-receipt-cards">{receiptItems.map(order=>{const meta=statusMeta[order.status];const loading=receiptLoadingId===order.id;return <article key={order.id}>
            <header><span className="row-icon r2"><Icon name="stock"/></span><div><b>{order.number}</b><small>{order.supplierName}</small></div><Status tone={meta.tone}>{meta.label}</Status></header>
            <dl><div><dt>Artículos</dt><dd>{order.itemCount}</dd></div><div><dt>Entrega</dt><dd>{order.expectedAt?formatRegionalCalendarDate(order.expectedAt,location?.country,{dateStyle:"short"}):"—"}</dd></div></dl>
            <footer><RowActionButton action="view" onClick={()=>openDetail(order.id,"view")}/>{canReceive&&<Button kind="success" icon="stock" disabled={loading} onClick={()=>void openReceipt(order.id)}>{loading?"Cargando…":order.status==="partially_received"?"Continuar recepción":"Registrar recepción"}</Button>}</footer>
          </article>})}</div>
        </>}
        {!receipts.isLoading&&!receipts.isError&&<Pagination page={page} size={size} total={receipts.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
        </>}
      </>:<>
        {suppliers.isLoading?<SupplierTableSkeleton/>
        :suppliers.isError?<PurchaseState icon="alert" title="No pudimos cargar los proveedores" text={suppliers.error.message} action={()=>suppliers.refetch()}/>
        :!supplierItems.length?<PurchaseState
          icon="truck"
          title={q||status?"Sin coincidencias":"Aún no hay proveedores"}
          text={q||status?"Ajusta la búsqueda o los filtros.":canManage?"Registra el primer proveedor para empezar a crear órdenes.":"No hay proveedores registrados."}
          action={canManage&&!q&&!status?()=>setSupplierDraft({...emptySupplier}):undefined}
        />:<>
          <div className="table-wrap hover-scroll purchases-table-wrap"><table>
            <thead><tr><th>PROVEEDOR</th><th>RUC</th><th>CONTACTO</th><th>ESTADO</th><th>ACCIONES</th></tr></thead>
            <tbody>{supplierItems.map((supplier,index)=><tr className={index%2?"alternate":""} key={supplier.id}>
              <td><span className={"row-icon r"+index%3}><Icon name="truck" size={18}/></span><b>{supplier.name}</b></td>
              <td>{supplier.taxId||"—"}</td>
              <td>{supplier.phone||supplier.email||"Sin contacto"}{supplier.phone&&supplier.email&&<small>{supplier.email}</small>}</td>
              <td><Status tone={supplier.active?"green":"gray"}>{supplier.active?"Activo":"Inactivo"}</Status></td>
              <td><div className="table-actions">{canManage&&<>
                <RowActionButton action="edit" onClick={()=>setSupplierDraft({id:supplier.id,taxId:supplier.taxId,name:supplier.name,email:supplier.email,phone:supplier.phone})}/>
                <RowActionButton action={supplier.active?"deactivate":"activate"} onClick={()=>setSupplierTarget(supplier)}/>
              </>}</div></td>
            </tr>)}</tbody>
          </table></div>

          <div className="management-cards purchases-cards">{supplierItems.map(supplier=><article key={supplier.id}>
            <header><span className="row-icon r1"><Icon name="truck"/></span><div><b>{supplier.name}</b><small>{supplier.taxId||"Sin RUC"}</small></div><Status tone={supplier.active?"green":"gray"}>{supplier.active?"Activo":"Inactivo"}</Status></header>
            <dl><div><dt>Teléfono</dt><dd>{supplier.phone||"—"}</dd></div><div><dt>Correo</dt><dd>{supplier.email||"—"}</dd></div></dl>
            {canManage&&<footer><RowActionButton action="edit" onClick={()=>setSupplierDraft({id:supplier.id,taxId:supplier.taxId,name:supplier.name,email:supplier.email,phone:supplier.phone})}/><RowActionButton action={supplier.active?"deactivate":"activate"} onClick={()=>setSupplierTarget(supplier)}/></footer>}
          </article>)}</div>
        </>}
        {!suppliers.isLoading&&!suppliers.isError&&<Pagination page={page} size={size} total={suppliers.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
      </>}
    </section>

    {orderDraft&&(orderDraftLoading||activeSuppliers.isLoading||inventory.isLoading
      ?<RemoteModalSkeleton className="purchase-order-modal" label="Cargando datos de compra" rows={7} close={()=>{setOrderDraft(null);setOrderDraftLoading(false)}}/>
      :activeSuppliers.isError||inventory.isError
        ?<PurchaseEditorError message={(activeSuppliers.error??inventory.error)?.message??"No pudimos cargar los datos necesarios."} close={()=>setOrderDraft(null)} retry={()=>{void activeSuppliers.refetch();void inventory.refetch()}}/>
        :<PurchaseOrderDialog initial={orderDraft} suppliers={activeSuppliers.data?.items??[]} inventory={inventory.data?.items??[]} currencySymbol={settings.currencySymbol} busy={saveOrder.isPending} close={()=>setOrderDraft(null)} save={draft=>saveOrder.mutate(draft)}/>
    )}

    {supplierDraft&&<SupplierDialog initial={supplierDraft} busy={supplierSave.isPending} close={()=>setSupplierDraft(null)} save={draft=>supplierSave.mutate(draft)}/>}

    {detailId&&(detail.isLoading
      ?<RemoteModalSkeleton className="purchase-detail-modal" label="Cargando orden de compra" rows={6} close={()=>setDetailId(null)}/>
      :detail.isError
        ?<PurchaseDetailError message={detail.error.message} close={()=>setDetailId(null)}/>
        :detail.data&&<PurchaseDetail
          order={detail.data}
          mode={detailMode}
          canManage={canManage}
          canApprove={canApprove}
          canReceive={canReceive}
          busy={transition.isPending||approve.isPending}
          currency={formatMoney(Number(detail.data.total),settings,location?.country)}
          country={location?.country}
          timezone={location?.timezone}
          close={()=>setDetailId(null)}
          edit={()=>{setDetailId(null);void editOrder(detail.data.id)}}
          changeStatus={next=>transition.mutate({id:detail.data.id,next})}
          approve={()=>approve.mutate(detail.data.id)}
          receive={()=>{setDetailId(null);changeTab("receipts")}}
          cancel={()=>setCancelTarget(detail.data)}
        />
    )}

    {receiptOrder&&<PurchaseReceiptDialog order={receiptOrder} busy={receive.isPending} close={()=>setReceiptOrder(null)} save={draft=>receive.mutate(draft)}/>}
    {receiptDetail&&<ReceiptHistoryDetail receipt={receiptDetail} canManage={canManage} close={()=>setReceiptDetail(null)} startReturn={()=>setReturnReceipt(receiptDetail)}/>}
    {returnReceipt&&<PurchaseReturnDialog receipt={returnReceipt} busy={returnPurchase.isPending} close={()=>setReturnReceipt(null)} save={draft=>returnPurchase.mutate(draft)}/>} 

    <ConfirmDialog
      open={Boolean(supplierTarget)}
      title={supplierTarget?.active?"Desactivar proveedor":"Activar proveedor"}
      description={supplierTarget?.active
        ?"“"+(supplierTarget?.name??"")+"” dejará de estar disponible para nuevas órdenes. Su historial se conservará."
        :"“"+(supplierTarget?.name??"")+"” volverá a estar disponible para nuevas órdenes."}
      tone={supplierTarget?.active?"danger":"success"}
      confirmLabel={supplierTarget?.active?"Desactivar":"Activar"}
      pending={supplierStatus.isPending}
      onCancel={()=>setSupplierTarget(null)}
      onConfirm={()=>supplierTarget&&supplierStatus.mutate(supplierTarget)}
    />

    <ConfirmDialog
      open={Boolean(cancelTarget)}
      title="Cancelar orden de compra"
      description={"“"+(cancelTarget?.number??"")+"” quedará cancelada y ya no podrá recibirse. El historial se conservará."}
      tone="danger"
      confirmLabel="Cancelar orden"
      pending={transition.isPending}
      onCancel={()=>setCancelTarget(null)}
      onConfirm={()=>cancelTarget&&transition.mutate({id:cancelTarget.id,next:"cancelled"})}
    />
  </>;
}

function PurchaseOrderDialog({initial,suppliers,inventory,currencySymbol,busy,close,save}:{initial:PurchaseOrderDraft;suppliers:Supplier[];inventory:PurchaseInventoryOption[];currencySymbol:string;busy:boolean;close:()=>void;save:(draft:PurchaseOrderDraft)=>void}){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const[createdItems,setCreatedItems]=useState<PurchaseInventoryOption[]>([]);
  const[itemTarget,setItemTarget]=useState<number|"new"|null>(null);
  const{control,register,handleSubmit,watch,setValue,formState:{errors,isSubmitted}}=useForm<PurchaseOrderDraft>({defaultValues:initial,resolver:purchaseOrderResolver,mode:"onSubmit",reValidateMode:"onChange"});
  const{fields,append,remove}=useFieldArray({control,name:"items"});
  const lines=watch("items");
  const catalog=[...inventory,...createdItems.filter(item=>!inventory.some(existing=>existing.id===item.id))];
  const total=lines.reduce((sum,line)=>sum+(Number(line.quantity)||0)*(Number(line.unitCost)||0),0);

  function applyItem(target:number|"new",item:PurchaseInventoryOption,preferredPresentationId?:string){
    const presentation=item.presentations.find(option=>option.id===preferredPresentationId)
      ??item.presentations.find(option=>option.presentationType==="unit")
      ??item.presentations[0];
    const next={inventoryItemId:item.id,presentationId:presentation?.id??"",quantity:"1",unitCost:""};
    if(target==="new")append(next);
    else{
      setValue(`items.${target}.inventoryItemId`,item.id,{shouldDirty:true,shouldValidate:isSubmitted});
      setValue(`items.${target}.presentationId`,presentation?.id??"",{shouldDirty:true,shouldValidate:isSubmitted});
    }
    setItemTarget(null);
  }

  const createItem=useMutation({
    mutationFn:({draft,file}:{target:number|"new";draft:Parameters<typeof createPurchaseInventoryItem>[0];file:File|null})=>createPurchaseInventoryItem(draft,file),
    onSuccess:(item,variables)=>{
      setCreatedItems(current=>current.some(existing=>existing.id===item.id)?current:[...current,item]);
      const factor=variables.draft.presentationType==="unit"?1:Number(variables.draft.unitsPerPresentation);
      const presentation=item.presentations.find(option=>
        option.presentationType===variables.draft.presentationType&&Number(option.unitsPerPresentation)===factor
      );
      applyItem(variables.target,item,presentation?.id);
      void qc.invalidateQueries({queryKey:["purchase-inventory"]});
      void qc.invalidateQueries({queryKey:["inventory"]});
      void qc.invalidateQueries({queryKey:["products"]});
      notify({tone:"success",title:"Artículo creado",message:item.name+" quedó con stock 0 y agregado a la orden."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo crear el artículo",message:error.message}),
  });

  return <>
    <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-order-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-order-title" aria-busy={busy}><div className="modal-accent"/>
      <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>ORDEN DE COMPRA</small><h2 id="purchase-order-title">{initial.id?"Editar orden":"Nueva orden de compra"}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
      <form onSubmit={handleSubmit(save)} noValidate inert={busy}><div className="purchase-form-body">
        <section className="purchase-form-section purchase-order-header-section">
          <div className="purchase-section-title"><span><Icon name="truck" size={17}/></span><div><b>Datos de la orden</b><small>Selecciona proveedor, fecha esperada y condiciones de compra.</small></div></div>
          <div className="form-grid purchase-order-meta-grid">
            <label>Proveedor<Select autoFocus {...register("supplierId")} aria-invalid={Boolean(errors.supplierId)}><option value="">{suppliers.length?"Selecciona un proveedor":"No hay proveedores activos"}</option>{suppliers.map(supplier=><option value={supplier.id} key={supplier.id}>{supplier.name}{supplier.taxId?" · "+supplier.taxId:""}</option>)}</Select>{errors.supplierId?.message&&<small className="wizard-field-error">{errors.supplierId.message}</small>}</label>
            <label>Entrega esperada<Input type="date" {...register("expectedAt")}/></label>
            <label className="span-2">Notas<Textarea rows={2} maxLength={500} {...register("notes")} placeholder="Condiciones, referencia o indicaciones para la compra"/>{errors.notes?.message&&<small className="wizard-field-error">{errors.notes.message}</small>}</label>
          </div>
        </section>

        <section className="purchase-form-section purchase-order-lines-section">
          <div className="purchase-section-title purchase-lines-title">
            <span><Icon name="stock" size={17}/></span>
            <div><b>Artículos solicitados</b><small>Define qué vas a pedir, en qué presentación, cantidad y costo.</small></div>
            {fields.length>0&&<Button type="button" kind="secondary" icon="plus" onClick={()=>setItemTarget("new")}>Agregar artículo</Button>}
          </div>

          {!fields.length?<div className="purchase-lines-empty">
            <p>Busca un artículo existente o crea uno nuevo para incluirlo en la orden.</p>
            <Button type="button" kind="secondary" icon="plus" onClick={()=>setItemTarget("new")}>Agregar artículo</Button>
          </div>:<div className="purchase-lines">{fields.map((field,index)=>{
            const value=lines[index];
            const selected=catalog.find(item=>item.id===value?.inventoryItemId);
            return <article className="purchase-line purchase-line-modern" key={field.id}>
              <div className="purchase-line-head">
                <span className="purchase-line-number">{index+1}</span>
                <span className="purchase-line-item-icon"><Icon name={selected?.kind==="ingredient"?"stock":"box"} size={16}/></span>
                <div className="purchase-line-item-copy">
                  <b>{selected?.name??"Artículo no disponible"}</b>
                  <small>{selected?(selected.kind==="ingredient"?"Insumo":"Producto vendible")+" · Unidad base: "+selected.unit:"Selecciona otro artículo"}</small>
                </div>
                <div className="purchase-line-actions" aria-label="Acciones del artículo">
                  <button type="button" className="purchase-line-remove" onClick={()=>remove(index)} aria-label={"Quitar "+(selected?.name??"artículo")}><Icon name="trash" size={14}/><span>Quitar</span></button>
                </div>
              </div>
              <div className="purchase-line-fields">
                <label>Presentación<Select {...register(`items.${index}.presentationId`)} disabled={!selected} aria-invalid={Boolean(errors.items?.[index]?.presentationId)}><option value="">Selecciona...</option>{(selected?.presentations??[]).map(presentation=><option value={presentation.id} key={presentation.id}>{presentationName(presentation.presentationType,presentation.unitsPerPresentation,selected?.unit)}</option>)}</Select>{errors.items?.[index]?.presentationId?.message&&<small className="wizard-field-error">{errors.items[index]?.presentationId?.message}</small>}</label>
                <label>Cantidad<Input type="number" min="0.001" step="0.001" inputMode="decimal" {...register(`items.${index}.quantity`)} aria-invalid={Boolean(errors.items?.[index]?.quantity)}/>{errors.items?.[index]?.quantity?.message&&<small className="wizard-field-error">{errors.items[index]?.quantity?.message}</small>}</label>
                <label>Costo unitario<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" {...register(`items.${index}.unitCost`)} placeholder="0.00" aria-invalid={Boolean(errors.items?.[index]?.unitCost)}/></div>{errors.items?.[index]?.unitCost?.message&&<small className="wizard-field-error">{errors.items[index]?.unitCost?.message}</small>}</label>
                <div className="purchase-line-total"><small>SUBTOTAL</small><b>{currencySymbol} {formatRegionalNumber((Number(value?.quantity)||0)*(Number(value?.unitCost)||0),undefined,{minimumFractionDigits:2,maximumFractionDigits:2})}</b></div>
              </div>
            </article>;
          })}</div>}
          {typeof errors.items?.message==="string"&&<div className="purchase-validation" role="alert"><Icon name="alert" size={15}/>{errors.items.message}</div>}
        </section>

        {fields.length>0&&<div className="purchase-total"><span><small>TOTAL ESTIMADO</small><b>{fields.length} {fields.length===1?"línea":"líneas"}</b></span><strong>{currencySymbol} {formatRegionalNumber(total,undefined,{minimumFractionDigits:2,maximumFractionDigits:2})}</strong></div>}
      </div><footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||!suppliers.length||fields.length===0}>{busy?"Guardando…":"Guardar borrador"}</Button></footer></form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </section></div>

    {itemTarget!==null&&<PurchaseItemDialog
      currencySymbol={currencySymbol}
      busy={createItem.isPending}
      close={()=>setItemTarget(null)}
      choose={item=>applyItem(itemTarget,item)}
      save={(draft,file)=>createItem.mutate({target:itemTarget,draft,file})}
    />}
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

function PurchaseDetail({order,mode,canManage,canApprove,canReceive,busy,currency,country,timezone,close,edit,changeStatus,approve,receive,cancel}:{order:PurchaseOrder;mode:"view"|"review";canManage:boolean;canApprove:boolean;canReceive:boolean;busy:boolean;currency:string;country?:string|null;timezone?:string|null;close:()=>void;edit:()=>void;changeStatus:(next:"draft"|"pending_approval"|"cancelled")=>void;approve:()=>void;receive:()=>void;cancel:()=>void}){
  const meta=statusMeta[order.status];
  const receivable=order.status==="approved"||order.status==="partially_received";
  const cancellable=order.status==="draft"||order.status==="pending_approval"||order.status==="approved";
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-detail-title" aria-busy={busy}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>{mode==="view"?"DETALLE DE ORDEN":"REVISAR ORDEN"}</small><h2 id="purchase-detail-title">{order.number}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header><div className="purchase-detail-body">
    <section className="purchase-detail-summary"><div><small>PROVEEDOR</small><b>{order.supplierName}</b></div><div><small>TOTAL</small><b>{currency}</b></div><div><small>ARTÍCULOS</small><b>{order.itemCount}</b></div><Status tone={meta.tone}>{meta.label}</Status></section>
    <section className="purchase-detail-meta"><div><small>CREADA</small><b>{formatRegionalDateTime(order.createdAt,{country,timeZone:timezone},{dateStyle:"medium",timeStyle:"short"})}</b></div><div><small>ENTREGA ESPERADA</small><b>{order.expectedAt?formatRegionalCalendarDate(order.expectedAt,country,{dateStyle:"medium"}):"Sin fecha"}</b></div><div><small>NOTAS</small><b>{order.notes||"Sin notas"}</b></div></section>
    <section className="purchase-detail-lines"><header><div><small>DETALLE</small><h3>Artículos de la orden</h3></div></header><div className="table-wrap hover-scroll"><table className="purchase-receipt-progress-table"><thead><tr><th>ARTÍCULO</th><th>PRESENTACIÓN</th><th>SOLICITADO</th><th>RECIBIDO</th><th>PENDIENTE</th><th>COSTO</th><th>SUBTOTAL</th></tr></thead><tbody>{order.items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}><td><b>{item.itemName}</b><small>{item.sku||item.unit}</small></td><td>{presentationName(item.presentationType,item.unitsPerPresentation,item.unit)}</td><td>{formatRegionalNumber(Number(item.quantity),country,{maximumFractionDigits:3})}</td><td><b className="purchase-received-qty">{formatRegionalNumber(Number(item.receivedQuantity),country,{maximumFractionDigits:3})}</b></td><td><b className={Number(item.pendingQuantity)>0?"purchase-pending-qty":""}>{formatRegionalNumber(Number(item.pendingQuantity),country,{maximumFractionDigits:3})}</b></td><td>{formatRegionalNumber(Number(item.unitCost),country,{minimumFractionDigits:2,maximumFractionDigits:4})}</td><td><b>{formatRegionalNumber(Number(item.lineTotal),country,{minimumFractionDigits:2,maximumFractionDigits:2})}</b></td></tr>)}</tbody></table></div></section>
    {mode==="review"&&(canManage||canApprove||canReceive)&&order.status!=="received"&&order.status!=="cancelled"&&<section className="purchase-detail-actions"><div><small>SIGUIENTE PASO</small><b>{nextStepLabel(order.status)}</b></div><div>{canManage&&order.status==="draft"&&<><Button kind="secondary" icon="edit" onClick={edit} disabled={busy}>Editar</Button><Button icon="arrowRightCircle" onClick={()=>changeStatus("pending_approval")} disabled={busy}>Enviar a aprobación</Button></>}{order.status==="pending_approval"&&<>{canManage&&<Button kind="ghost" icon="chevronLeft" onClick={()=>changeStatus("draft")} disabled={busy}>Volver a borrador</Button>}{canApprove&&<Button icon="check" onClick={approve} disabled={busy}>Aprobar</Button>}</>}{canReceive&&receivable&&<Button kind="success" icon="stock" onClick={receive} disabled={busy}>{order.status==="partially_received"?"Continuar en Recepciones":"Ir a Recepciones"}</Button>}{canManage&&cancellable&&<Button kind="danger" icon="close" onClick={cancel} disabled={busy}>Cancelar orden</Button>}</div></section>}
  </div>{busy&&<div className="modal-busy" role="status"><i/><span>Procesando…</span></div>}</section></div>;
}
function ReceiptHistoryDetail({receipt,canManage,close,startReturn}:{receipt:PurchaseReceiptDetail;canManage:boolean;close:()=>void;startReturn:()=>void}){
 const returnable=receipt.items.some(i=>Number(i.returnableQuantity)>0);
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-detail-modal modal-panel-in" role="dialog" aria-modal="true">
  <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>RECEPCIÓN HISTÓRICA</small><h2>{receipt.code}</h2></div><button onClick={close} aria-label="Cerrar"><Icon name="close"/></button></header>
  <div className="purchase-detail-body"><section className="purchase-detail-summary"><div><small>ORDEN</small><b>{receipt.number}</b></div><div><small>PROVEEDOR</small><b>{receipt.supplierName}</b></div><div><small>USUARIO</small><b>{receipt.createdByName}</b></div></section>
   <div className="table-wrap"><table><thead><tr><th>ARTÍCULO</th><th>RECIBIDO</th><th>DEVUELTO/CORREGIDO</th><th>DISPONIBLE</th><th>COSTO</th></tr></thead><tbody>{receipt.items.map(i=><tr key={i.id}><td><b>{i.itemName}</b></td><td>{i.quantity}</td><td>{i.returnedQuantity}</td><td>{i.returnableQuantity}</td><td>{i.unitCost}</td></tr>)}</tbody></table></div>
   {receipt.notes&&<p>{receipt.notes}</p>}
  </div>
  <footer><Button kind="ghost" onClick={close}>Cerrar</Button>{canManage&&returnable&&<Button kind="danger" icon="truck" onClick={startReturn}>Corregir / devolver</Button>}</footer>
 </section></div>
}

function PurchaseDetailError({message,close}:{message:string;close:()=>void}){return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-detail-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="alert"/></span><div><small>ORDEN DE COMPRA</small><h2>No pudimos cargar el detalle</h2></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><PurchaseState icon="alert" title="Detalle no disponible" text={message}/></section></div>}

function PurchaseEditorError({message,close,retry}:{message:string;close:()=>void;retry:()=>void}){return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal purchase-order-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="alert"/></span><div><small>ORDEN DE COMPRA</small><h2>No pudimos preparar el formulario</h2></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><PurchaseState icon="alert" title="Catálogos no disponibles" text={message} action={retry}/></section></div>}

function presentationName(type:string,factor:string,unit?:string){if(type==="unit")return `Unidad base · ${unit??"und"}`;return `${type==="box"?"Caja":"Paquete"} x ${formatRegionalNumber(Number(factor),undefined,{maximumFractionDigits:3})} ${unit??"und"}`}
function nextStepLabel(status:PurchaseStatus){if(status==="draft")return"Completa la orden y envíala para aprobación.";if(status==="pending_approval")return"Revisa el total y aprueba antes de recibir.";if(status==="approved")return"Registra lo que realmente llegó; la orden no movió stock.";if(status==="partially_received")return"Completa las cantidades que aún están pendientes.";return"Sin acciones pendientes."}
function formatMoney(amount:number,settings:{currencySymbol:string;currencyPosition:"before"|"after";currencyDecimals:number},country?:string|null){const value=formatRegionalNumber(amount,country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});return settings.currencyPosition==="before"?`${settings.currencySymbol} ${value}`:`${value} ${settings.currencySymbol}`}

function PurchaseState({icon,title,text,action}:{icon:"alert"|"receipt"|"truck";title:string;text:string;action?:()=>void}){return <div className="catalog-state purchases-state"><span><Icon name={icon}/></span><b>{title}</b><p>{text}</p>{action&&<Button kind="secondary" icon={icon==="alert"?"refresh":"plus"} onClick={action}>{icon==="alert"?"Reintentar":icon==="truck"?"Nuevo proveedor":"Nueva orden"}</Button>}</div>}
function PurchaseTableSkeleton(){return <div className="purchases-loading" aria-label="Cargando compras">{Array.from({length:6},(_,row)=><i key={row}/>)}</div>}
function SupplierTableSkeleton(){return <div className="purchases-loading" aria-label="Cargando proveedores">{Array.from({length:6},(_,row)=><i key={row}/>)}</div>}
