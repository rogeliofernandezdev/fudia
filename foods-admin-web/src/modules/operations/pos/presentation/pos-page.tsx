"use client";
import {DataTable} from "@/design-system/data-table";
import {canChargeAccount} from "../../orders/domain/service-flow";
import {Dialog} from "@/design-system/dialog";
import "./pos.css";
import Link from "next/link";
import dynamic from "next/dynamic";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,Icon,IconName,PageHeader,Pagination,RowActionButton,SearchField,Select,Status,ModalHeader,ModalFooter,ModalWorkspace,ModalItemList,ModalItemNote,ModalItemComponents,DetailContact,DetailTimeline,Skeleton,Tabs} from "@/design-system";
import {useFeedback,useMoney,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {pageRoutes} from "@/shared/routing/page-routes";
import {cashShiftAttribution} from "../../cash/domain/shift-attribution";
import {getCurrentCashShift} from "../../cash/infrastructure/cash-api";
import type {Payment,PaymentDraft,PaymentMethodOption,POSOrderDetail,POSOrderSummary} from "../domain/types";
import {createPayment,getPOSOrder,listPOSOrders,listSalesPaymentMethods,refundPayment} from "../infrastructure/pos-api";
const PaymentDialog=dynamic(()=>import("./pos-dialogs").then(module=>module.PaymentDialog),{ssr:false});
const RefundDialog=dynamic(()=>import("./pos-dialogs").then(module=>module.RefundDialog),{ssr:false});
const PaymentTicketDialog=dynamic(()=>import("@/modules/sales").then(module=>module.PaymentTicketDialog),{ssr:false});

const paymentMeta={
  pending:{label:"Sin pagos",tone:"gray" as const},
  partial:{label:"Pago parcial",tone:"orange" as const},
  paid:{label:"Pagado",tone:"green" as const},
};
const channelLabel:Record<string,string>={salon:"Salón",mostrador:"Recojo en local",recojo:"Recojo en local",delivery:"Delivery",whatsapp:"WhatsApp"};
const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"box",recojo:"box",delivery:"truck",whatsapp:"share"};

export function POSPage({initialOrderId=""}:{initialOrderId?:string}){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const money=useMoney();
  const canManage=can("cash.manage");
  const[q,setQ]=useState("");
  const debouncedQ=useDebouncedValue(q);
  const[paymentStatus,setPaymentStatus]=useState("all");
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[paymentOrderId,setPaymentOrderId]=useState<string|null>(initialOrderId||null);
  const[detailId,setDetailId]=useState<string|null>(null);
  const[refundTarget,setRefundTarget]=useState<Payment|null>(null);
  const[ticketTarget,setTicketTarget]=useState<{orderId:string;paymentId?:string}|null>(null);

  const current=useQuery({
    queryKey:["cash-shift","current"],
    queryFn:getCurrentCashShift,
    refetchInterval:30000,
  });
  const methods=useQuery({
    queryKey:["payment-methods","sales"],
    queryFn:listSalesPaymentMethods,
    enabled:Boolean(paymentOrderId),
    staleTime:60000,
  });
  const orders=useQuery({
    queryKey:["pos-orders",debouncedQ,paymentStatus,page,size],
    queryFn:()=>listPOSOrders({q:debouncedQ,paymentStatus,page,pageSize:size}),
    refetchInterval:30000,
  });
  const detail=useQuery({
    queryKey:["pos-order",detailId],
    queryFn:()=>getPOSOrder(detailId!),
    enabled:Boolean(detailId),
  });
  const paymentOrder=useQuery({
    queryKey:["pos-order",paymentOrderId],
    queryFn:()=>getPOSOrder(paymentOrderId!),
    enabled:Boolean(paymentOrderId)&&canManage,
    staleTime:0,
    refetchOnMount:"always",
    refetchOnWindowFocus:false,
  });

  function refresh(){
    void qc.invalidateQueries({queryKey:["pos-orders"]});
    void qc.invalidateQueries({queryKey:["pos-order"]});
    for(const key of ["salon-floor","orders","order"])void qc.invalidateQueries({queryKey:[key]});
    void qc.invalidateQueries({queryKey:["cash-registers"]});
    void qc.invalidateQueries({queryKey:["cash-shift"]});
    void qc.invalidateQueries({queryKey:["sales"]});
    void qc.invalidateQueries({queryKey:["sale-detail"]});
    void qc.invalidateQueries({queryKey:["dashboard"]});
  }

  const pay=useMutation({
    retry:false,
    mutationFn:({orderId,draft}:{orderId:string;draft:Parameters<typeof createPayment>[1]})=>createPayment(orderId,draft),
    onSuccess:(item,variables)=>{
      setPaymentOrderId(null);
      refresh();
      notify({tone:"success",title:"Cobro registrado",message:"El pago quedó asociado al turno activo.",onClose:()=>setTicketTarget({orderId:variables.orderId,paymentId:item.id})});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo registrar el cobro",message:error.message}),
  });

  const refund=useMutation({
    mutationFn:({paymentId,draft}:{paymentId:string;draft:Parameters<typeof refundPayment>[1]})=>refundPayment(paymentId,draft),
    onSuccess:()=>{
      setRefundTarget(null);
      refresh();
      notify({tone:"success",title:"Devolución registrada",message:"El saldo del pedido y la trazabilidad del turno quedaron actualizados."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo registrar la devolución",message:error.message}),
  });

  const shift=current.data?.shift??null;
  const items=orders.data?.items??[];

  function dateTime(value:string){
    return formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  }
  function actionsFor(item:POSOrderSummary){
    return <div className="pos-actions">
      <RowActionButton action="view" label={`Ver detalle de ${item.code}`} onClick={()=>setDetailId(item.id)}/>
      {canManage&&item.paymentStatus!=="paid"&&<RowActionButton action="charge" label={shift?`Cobrar ${item.code}`:"Abre un turno para cobrar"} disabled={!shift||!canChargeAccount({...item,billClosedAt:item.billClosed?"closed":undefined})} onClick={()=>setPaymentOrderId(item.id)}/>}
    </div>;
  }

  return <div className="pos-page">
    <PageHeader title="Punto de venta" description="Cobra pedidos existentes y mantiene Caja sincronizada con pagos y devoluciones."/>

    <section className={"pos-shift-banner "+(shift?"active":"missing")}>
      <span className="pos-shift-icon"><Icon name={shift?"register":"alert"} size={20}/></span>
      {current.isLoading?<div className="pos-shift-loading" aria-label="Cargando turno" aria-busy="true"><i/><i/></div>
      :shift?<><div className="pos-shift-copy"><small>Turno activo</small><b>{shift.cashRegisterName}</b><p>Cajero: {cashShiftAttribution(shift).name}</p></div><Status tone="green">Listo para cobrar</Status></>
      :<><div><small>Turno requerido</small><b>No estás asignado a una caja abierta</b><p>Inicia o únete a un turno antes de registrar cobros o devoluciones.</p></div>{cashLink()}</>}
    </section>

    <section className="panel standardized-management pos-panel">
      <div className="pos-toolbar">
        <SearchField value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder="Buscar pedido, cliente o mesa…" label="Buscar pedidos por cliente o mesa"/>
        <Select chip aria-label="Filtrar por estado de cobro" value={paymentStatus} onChange={event=>{setPaymentStatus(event.target.value);setPage(1)}}>
          <option value="all">Todos</option>
          <option value="pending">Sin pagos</option>
          <option value="partial">Pago parcial</option>
          <option value="paid">Pagado</option>
        </Select>
      </div>

      {orders.isLoading?<POSLoading/>
      :orders.isError?<ErrorState title="No pudimos cargar los pedidos" text={orders.error.message} onRetry={()=>void orders.refetch()}/>
      :!items.length?<EmptyState icon="receipt" title={q?"Sin coincidencias":paymentStatus==="paid"?"Aún no hay pedidos pagados":paymentStatus==="all"?"Aún no hay pedidos":"No hay pedidos con ese estado de cobro"} text={q?"Prueba con otro término de búsqueda.":"Los pedidos aparecerán aquí según su estado de cobro."}/>
      :<DataTable label="Pedidos para cobrar" containerClassName="hover-scroll pos-table-wrap" className="pos-table">
        <thead><tr><th>Pedido</th><th>Canal</th><th className="pos-money">Total</th><th className="pos-money">Pagado</th><th className="pos-money">Saldo</th><th>Cobro</th><th className="pos-th-actions">Acciones</th></tr></thead>
        <tbody>{items.map((item,index)=>{
          const meta=paymentMeta[item.paymentStatus];
          const subject=item.tableName||item.customerName||item.code;
          return <tr className={index%2?"alternate":""} key={item.id}>
            <td><span className="row-icon pos-row-icon"><Icon name="receipt" size={18}/></span><b>{subject}</b></td>
            <td><span className="pos-channel-cell"><Icon name={channelIcons[item.channel]??"receipt"} size={14}/>{channelLabel[item.channel]??item.channel}</span></td>
            <td className="pos-money"><b>{money(Number(item.total))}</b></td>
            <td className="pos-money">{money(Number(item.paidAmount))}</td>
            <td className="pos-money"><strong className="pos-balance">{money(Number(item.remainingAmount))}</strong></td>
            <td><Status tone={meta.tone}>{meta.label}</Status></td>
            <td>{actionsFor(item)}</td>
          </tr>;
        })}</tbody>
      </DataTable>}

      {!orders.isLoading&&!orders.isError&&items.length>0&&<div className="management-cards pos-order-cards">{items.map(item=>{
        const meta=paymentMeta[item.paymentStatus];
        const subject=item.tableName||item.customerName||item.code;
        return <article key={item.id}>
          <header><span className="row-icon pos-row-icon"><Icon name="receipt" size={18}/></span><div><b>{subject}</b></div><Status tone={meta.tone}>{meta.label}</Status></header>
          <div className="pos-card-channel"><Icon name={channelIcons[item.channel]??"receipt"} size={14}/><span>{channelLabel[item.channel]??item.channel}</span></div>
          <dl><div><dt>Total</dt><dd>{money(Number(item.total))}</dd></div><div><dt>Pagado</dt><dd>{money(Number(item.paidAmount))}</dd></div><div className="remaining"><dt>Saldo</dt><dd>{money(Number(item.remainingAmount))}</dd></div></dl>
          <footer>{actionsFor(item)}</footer>
        </article>;
      })}</div>}

      {!orders.isLoading&&!orders.isError&&<Pagination page={page} size={size} total={orders.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
    </section>

    {paymentOrderId&&<POSPaymentEntry
      loading={paymentOrder.isPending||paymentOrder.isFetching}
      error={paymentOrder.error?.message}
      data={paymentOrder.data}
      shiftLoading={current.isLoading}
      shiftError={current.error?.message}
      shiftName={shift?shift.cashRegisterName:null}
      methodsLoading={methods.isLoading}
      methodsError={methods.error?.message}
      methods={methods.data?.items}
      retryMethods={()=>void methods.refetch()}
      canManage={canManage}
      busy={pay.isPending}
      formatMoney={money}
      close={()=>{if(!pay.isPending)setPaymentOrderId(null)}}
      retryOrder={()=>void paymentOrder.refetch()}
      retryShift={()=>void current.refetch()}
      save={draft=>{if(canManage&&shift&&!pay.isPending)pay.mutate({orderId:paymentOrderId,draft})}}
    />}
    {refundTarget&&shift&&<RefundDialog payment={refundTarget} busy={refund.isPending} formatMoney={money} close={()=>setRefundTarget(null)} save={draft=>refund.mutate({paymentId:refundTarget.id,draft})}/>}
    {detailId&&<POSDetailDialog
      loading={detail.isLoading}
      error={detail.error?.message}
      data={detail.data}
      canManage={canManage}
      hasShift={Boolean(shift)}
      formatMoney={money}
      formatDateTime={dateTime}
      retry={()=>void detail.refetch()}
      close={()=>setDetailId(null)}
      refund={setRefundTarget}
      ticket={()=>{setDetailId(null);setTicketTarget({orderId:detailId})}}
    />}
    {ticketTarget&&<PaymentTicketDialog {...ticketTarget} close={()=>setTicketTarget(null)}/>}
  </div>;
}

function POSPaymentEntry({loading,error,data,shiftLoading,shiftError,shiftName,methodsLoading,methodsError,methods,retryMethods,canManage,busy,formatMoney,close,retryOrder,retryShift,save}:{loading:boolean;error?:string;data?:POSOrderDetail;shiftLoading:boolean;shiftError?:string;shiftName:string|null;methodsLoading:boolean;methodsError?:string;methods?:PaymentMethodOption[];retryMethods:()=>void;canManage:boolean;busy:boolean;formatMoney:(value:number)=>string;close:()=>void;retryOrder:()=>void;retryShift:()=>void;save:(draft:PaymentDraft)=>void}){
  let state:{title:string;text:string;icon?:IconName;retry?:()=>void;cash?:boolean}|undefined;
  if(!canManage)state={title:"No puedes registrar cobros",text:"Tu rol necesita permiso para administrar cobros.",icon:"lock"};
  else if(error)state={title:"No pudimos cargar el pedido",text:error,retry:retryOrder};
  else if(shiftError)state={title:"No pudimos validar tu caja",text:shiftError,retry:retryShift};
  else if(methodsError)state={title:"No pudimos cargar los medios de pago",text:methodsError,retry:retryMethods};
  else if(!loading&&!shiftLoading&&!methodsLoading){
    if(!data)state={title:"Pedido no disponible",text:"No pudimos encontrar este pedido en el local activo.",retry:retryOrder};
    else if(data.paymentStatus==="paid"||Number(data.remainingAmount)<=0)state={title:"El pedido ya está pagado",text:"No tiene saldo pendiente de cobro.",icon:"circleCheck"};
    else if(data.order.completedAt||data.order.status==="cancelado")state={title:"El pedido está cerrado",text:"No se pueden registrar nuevos cobros.",icon:"lock"};
    else if(!canChargeAccount(data.order))state=data.order.channel==="salon"?{title:"Cuenta no disponible para cobro",text:"El mozo debe entregar todos los productos y cerrar la cuenta antes de cobrar.",icon:"clock"}:{title:"Pedido no listo",text:"El pedido todavía no está listo para cobrar.",icon:"clock"};
    else if(!shiftName)state={title:"Necesitas un turno de caja",text:"Abre un turno o únete a una caja para registrar el cobro.",icon:"register",cash:true};
    else if(!methods?.length)state={title:"Sin medios de pago para ventas",text:"Activa al menos un medio para ventas en Configuración › Medios de pago.",icon:"wallet"};
    else return <PaymentDialog methods={methods} order={{...data.order,paidAmount:data.paidAmount,remainingAmount:data.remainingAmount,paymentStatus:data.paymentStatus}} shiftName={shiftName} busy={busy} formatMoney={formatMoney} close={close} save={save}/>;
  }
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal pos-payment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-payment-entry-title">

    <ModalHeader title={data?.order.tableName||data?.order.customerName||"Cobrar saldo"} titleId="pos-payment-entry-title" icon="cash" subtitle="Cobro" onClose={close} busy={busy} closeLabel="Cerrar"/>
    {state?state.retry?<ErrorState title={state.title} text={state.text} onRetry={state.retry}/>:<EmptyState icon={state.icon} title={state.title} text={state.text} action={state.cash?cashLink():undefined}/>:<POSPaymentLoading/>}
  </Dialog></div>;
}

function POSPaymentLoading(){return <div className="pos-payment-body pos-payment-loading" aria-label="Cargando formulario de cobro" aria-busy="true">
  <div className="pos-payment-overview"><i/><i/></div><i/>
  <div className="pos-payment-loading-methods">{Array.from({length:4},(_,index)=><i key={index}/>)}</div>
  <div className="pos-payment-loading-field"><i/><b/></div>
  <div className="pos-payment-loading-actions"><i/><i/></div>
</div>}

/** Detalle de cobro con el patrón de Detalle de pedido: cabecera con mesa o cliente, identidad, tarjeta, pestañas y resumen fijo. */
export function POSDetailDialog({loading,error,data,canManage,hasShift,formatMoney,formatDateTime,retry,close,refund,ticket}:{loading:boolean;error?:string;data?:POSOrderDetail;canManage:boolean;hasShift:boolean;formatMoney:(value:number)=>string;formatDateTime:(value:string)=>string;retry?:()=>void;close:()=>void;refund:(payment:Payment)=>void;ticket?:()=>void}){
  const closed=Boolean(data&&(data.order.completedAt||data.order.status==="cancelado"));
  const tableName=data?.order.tableName?.trim();
  const title=tableName?(/^mesa\b/i.test(tableName)?tableName:`Mesa ${tableName}`):data?.order.customerName||(data?"Venta directa":"Detalle de cobro");
  const icon:IconName=tableName?"utensils":data?channelIcons[data.order.channel]??"receipt":"receipt";
  const paid=Number(data?.paidAmount??0);
  const remaining=Number(data?.remainingAmount??0);
  const[tab,setTab]=useState<"products"|"payments">("products");
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal pos-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-detail-title" aria-busy={loading}>
    {loading?<POSDetailLoading close={close} withFooter={Boolean(ticket)}/>:<>
    <ModalHeader title={title} titleId="pos-detail-title" icon={icon} onClose={close} closeLabel="Cerrar detalle"/>
    {error?<div className="modal-body pos-detail-body"><ErrorState title="No pudimos cargar el cobro" text={error} onRetry={retry}/></div>
    :!data?<div className="modal-body pos-detail-body"><EmptyState icon="receipt" title="No hay detalle disponible"/></div>
    :<ModalWorkspace summary={<section className={`modal-summary pos-detail-totals${remaining<=0.00001?" is-paid":""}`} aria-labelledby="pos-detail-summary-title">
      <div className="modal-summary-heading"><h3 id="pos-detail-summary-title" className="modal-section-title">Resumen de cobro</h3></div>
      <div className="modal-summary-lines">
        <div className="modal-summary-line"><span>Total del pedido</span><b>{formatMoney(Number(data.order.total))}</b></div>
        <div className={`modal-summary-line pos-detail-paid${paid>0.00001?" has-payment":""}`}><span>Pagado neto</span><b>{formatMoney(paid)}</b></div>
      </div>
      <div className="modal-summary-total"><span>{remaining>0.00001?"Saldo pendiente":"Total pagado"}</span><strong>{formatMoney(remaining>0.00001?remaining:Number(data.order.total))}</strong></div>
    </section>}>
    <div className="modal-body pos-detail-body" tabIndex={0} aria-label="Productos y pagos del pedido">
      <div className="pos-detail-content">
        <section className="pos-detail-hero" aria-label="Resumen del cobro">
          <div className="pos-detail-identity">
            <div className="pos-detail-status" aria-label="Estado de pago"><Status tone={paymentMeta[data.paymentStatus].tone}>{paymentMeta[data.paymentStatus].label}</Status><span>{data.order.code}</span></div>
            <p><Icon name="clock" size={14}/><time dateTime={data.order.createdAt}>{formatDateTime(data.order.createdAt)}</time></p>
            <p><Icon name={channelIcons[data.order.channel]??"receipt"} size={14}/><span>{[channelLabel[data.order.channel]??data.order.channel,data.order.channel==="salon"&&`Atendido por ${data.order.waiterName||"sin asignar"}`].filter(Boolean).join(" · ")}</span></p>
          </div>
          {data.order.customerName&&<DetailContact name={data.order.customerName}/>}
        </section>

        <Tabs className="pos-detail-tabs" label="Información del cobro" value={tab} onChange={setTab} items={[
          {value:"products",label:"Productos",icon:"dish",count:data.order.items?.length??0},
          {value:"payments",label:"Pagos",icon:"payment",count:data.payments.length},
        ]}/>

        <section className="detail-panel pos-detail-panel" role="tabpanel" aria-labelledby="pos-detail-products-title" hidden={tab!=="products"}>
          <h3 id="pos-detail-products-title" className="sr-only">Productos del pedido</h3>
          {data.order.items?.length?<ModalItemList>{data.order.items.map(item=><div className="modal-item-row" role="listitem" key={item.id}><b className="modal-item-quantity">{Number(item.qty)}×</b><div className="modal-item-info"><span className="modal-item-name">{item.name}</span>{item.note&&<ModalItemNote>{item.note}</ModalItemNote>}<span className="modal-item-meta"><span>{formatMoney(Number(item.unitPrice))} c/u</span></span><ModalItemComponents label={`Detalle de ${item.name}`} items={[...(item.selections??[]).map((selection,index)=>({key:`selection-${index}`,group:selection.groupName,name:selection.name,extra:Number(selection.surcharge)>0&&<small>+{formatMoney(Number(selection.surcharge))}</small>})),...(item.modifiers??[]).map((modifier,index)=>({key:`modifier-${index}`,group:modifier.groupName,name:modifier.name}))]}/></div><strong className="modal-item-total">{formatMoney(Number(item.qty)*Number(item.unitPrice))}</strong></div>)}</ModalItemList>:<p className="pos-detail-empty">No hay productos registrados.</p>}
        </section>
        <section className="detail-panel pos-detail-panel" role="tabpanel" aria-labelledby="pos-detail-payments-title" hidden={tab!=="payments"}>
          <h3 id="pos-detail-payments-title" className="sr-only">Pagos registrados</h3>
          {data.payments.length?<DetailTimeline label="Pagos del pedido" steps={data.payments.map(payment=>{
            const net=Number(payment.netAmount);
            return {
              key:payment.id,
              icon:payment.affectsCash?"cash":"wallet",
              title:payment.methodName,
              time:formatDateTime(payment.createdAt),
              description:[payment.cashRegisterName,`Cobrado por: ${payment.createdByName||"Sin información"}`,payment.reference].filter(Boolean).join(" · "),
              aside:<span className="pos-payment-values">{formatMoney(Number(payment.amount))}{Number(payment.refundedAmount)>0&&<small>Devuelto {formatMoney(Number(payment.refundedAmount))}</small>}{canManage&&net>0&&!closed&&<Button kind="ghost" className="pos-refund-action" icon="undo" disabled={!hasShift} onClick={()=>refund(payment)}>Devolver</Button>}</span>,
            };
          })}/>
          :<p className="pos-detail-empty">Sin pagos registrados.</p>}
        </section>
      </div>
    </div></ModalWorkspace>}
    {!error&&data&&paid>0&&ticket&&<ModalFooter><Button kind="secondary" icon="eye" onClick={ticket}>Ver resumen de pago</Button></ModalFooter>}
    </>}
  </Dialog></div>;
}

function POSLoading(){return <>
  <div className="table-wrap pos-table-wrap pos-table-skeleton" aria-label="Cargando pedidos" aria-busy="true"><table className="pos-table"><thead><tr><th>Pedido</th><th>Canal</th><th>Total</th><th>Pagado</th><th>Saldo</th><th>Cobro</th><th>Acciones</th></tr></thead><tbody>{Array.from({length:5},(_,index)=><tr key={index}><td><span className="pos-sk-entity"><i/><span><b/></span></span></td>{Array.from({length:5},(_,cell)=><td key={cell}><i className="pos-sk-line"/></td>)}<td><span className="pos-sk-actions"><i/><i/></span></td></tr>)}</tbody></table></div>
  <div className="management-cards pos-order-cards pos-card-skeleton" aria-hidden="true">{Array.from({length:4},(_,index)=><article key={index}><header><i/><span><b/></span><i/></header><i/><div/><footer><i/><i/></footer></article>)}</div>
</>}

/** Skeleton con la geometría final: cabecera, identidad, tarjeta, pestañas, productos, resumen y acción. */
function POSDetailLoading({close,withFooter}:{close:()=>void;withFooter:boolean}){return <>
  <ModalHeader title={<><span className="sr-only">Cargando detalle de cobro</span><Skeleton width="var(--size-138)" height="var(--size-16)"/></>} titleId="pos-detail-title"
    iconSlot={<Skeleton width="var(--size-44)" height="var(--size-44)" radius="var(--radius-tile)"/>} onClose={close} closeLabel="Cerrar detalle"/>
  <ModalWorkspace summary={<section className="modal-summary" aria-hidden="true">
    <div className="modal-summary-heading"><Skeleton width="var(--size-138)" height="var(--size-13)"/></div>
    <div className="modal-summary-lines">{Array.from({length:2},(_,index)=><div className="modal-summary-line" key={index}><Skeleton width="var(--size-72)" height="var(--size-9)"/><Skeleton width="var(--size-68)" height="var(--size-16)"/></div>)}</div>
    <div className="modal-summary-total"><Skeleton width="var(--size-116)" height="var(--size-10)"/><Skeleton width="var(--size-108)" height="var(--size-24)"/></div>
  </section>}>
  <div className="modal-body pos-detail-body" role="status" aria-label="Cargando detalle del pedido" aria-busy="true">
    <div className="pos-detail-content" aria-hidden="true">
      <div className="pos-detail-hero">
        <div className="pos-detail-identity"><Skeleton width="var(--size-72)" height="var(--size-24)" radius="var(--radius-pill)"/><Skeleton width="var(--size-150)" height="var(--size-9)"/></div>
        <Skeleton width="100%" height="var(--size-68)" radius="var(--radius-card)"/>
      </div>
      <Skeleton width="min(var(--size-320),100%)" height="var(--size-44)" radius="var(--radius-pill)"/>
      <ModalItemList className="pos-detail-panel">
        {Array.from({length:3},(_,index)=><div className="modal-item-row" role="listitem" key={index}>
          <Skeleton width="var(--size-32)" height="var(--size-32)" radius="var(--radius-pill)"/>
          <span className="pos-detail-skeleton-copy"><Skeleton width="min(var(--size-220),72%)" height="var(--size-11)"/><Skeleton width="var(--size-92)" height="var(--size-9)"/></span>
          <Skeleton width="var(--size-66)" height="var(--size-10)"/>
        </div>)}
      </ModalItemList>
    </div>
  </div>
  </ModalWorkspace>
  {withFooter&&<ModalFooter><Skeleton className="pos-detail-skeleton-action"/></ModalFooter>}
</>}

/** Acceso a Caja cuando falta turno: enlace con apariencia de botón secundario. */
function cashLink(){return <Link href={pageRoutes.cash} className="button secondary"><Icon name="register" size={18}/><span>Ir a Caja</span></Link>}
