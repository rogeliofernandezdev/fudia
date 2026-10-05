"use client";
import "./pos.css";
import Link from "next/link";
import dynamic from "next/dynamic";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,IconName,Input,PageHeader,Pagination,RowActionButton,Select,Status} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {pageRoutes} from "@/shared/routing/page-routes";
import {cashShiftAttribution} from "../../cash/domain/shift-attribution";
import {getCurrentCashShift} from "../../cash/infrastructure/cash-api";
import type {Payment,POSOrderDetail,POSOrderSummary} from "../domain/types";
import {paymentMethodMeta} from "./pos-meta";
import {createPayment,getPOSOrder,listPOSOrders,refundPayment} from "../infrastructure/pos-api";
const PaymentDialog=dynamic(()=>import("./pos-dialogs").then(module=>module.PaymentDialog),{ssr:false});
const RefundDialog=dynamic(()=>import("./pos-dialogs").then(module=>module.RefundDialog),{ssr:false});

const paymentMeta={
  pending:{label:"Sin pagos",tone:"gray" as const},
  partial:{label:"Pago parcial",tone:"orange" as const},
  paid:{label:"Pagado",tone:"green" as const},
};
const channelLabel:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};
const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"store",recojo:"box",delivery:"truck",whatsapp:"share"};

export function POSPage({initialOrderId=""}:{initialOrderId?:string}){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const settings=useSettings();
  const canManage=can("cash.manage");
  const[q,setQ]=useState("");
  const debouncedQ=useDebouncedValue(q);
  const[paymentStatus,setPaymentStatus]=useState("unpaid");
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[paymentTarget,setPaymentTarget]=useState<POSOrderSummary|null>(null);
  const[detailId,setDetailId]=useState<string|null>(initialOrderId||null);
  const[refundTarget,setRefundTarget]=useState<Payment|null>(null);

  const current=useQuery({
    queryKey:["cash-shift","current"],
    queryFn:getCurrentCashShift,
    refetchInterval:30000,
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

  function refresh(){
    void qc.invalidateQueries({queryKey:["pos-orders"]});
    void qc.invalidateQueries({queryKey:["pos-order"]});
    void qc.invalidateQueries({queryKey:["cash-registers"]});
    void qc.invalidateQueries({queryKey:["cash-shift"]});
  }

  const pay=useMutation({
    mutationFn:({orderId,draft}:{orderId:string;draft:Parameters<typeof createPayment>[1]})=>createPayment(orderId,draft),
    onSuccess:(item)=>{
      setPaymentTarget(null);
      refresh();
      notify({tone:"success",title:"Cobro registrado",message:item.method==="cash"?"El pago quedó reflejado también en Caja.":"El pago quedó asociado al turno activo."});
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

  function money(value:number){
    if(!Number.isFinite(value))return"—";
    const formatted=formatRegionalNumber(Math.abs(value),location?.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
    const sign=value<0?"-":"";
    return settings.currencyPosition==="before"?sign+settings.currencySymbol+" "+formatted:sign+formatted+" "+settings.currencySymbol;
  }
  function dateTime(value:string){
    return formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  }
  function actionsFor(item:POSOrderSummary){
    return <div className="pos-actions">
      <RowActionButton action="view" label={`Ver detalle de ${item.code}`} onClick={()=>setDetailId(item.id)}/>
      {canManage&&item.paymentStatus!=="paid"&&<RowActionButton action="charge" label={shift?`Cobrar ${item.code}`:"Abre un turno para cobrar"} disabled={!shift} onClick={()=>setPaymentTarget(item)}/>} 
    </div>;
  }

  return <div className="pos-page">
    <PageHeader eyebrow="OPERACIÓN" title="Punto de venta" description="Cobra pedidos existentes y mantiene Caja sincronizada con pagos y devoluciones."/>

    <section className={"pos-shift-banner "+(shift?"active":"missing")}>
      <span className="pos-shift-icon"><Icon name={shift?"register":"alert"} size={19}/></span>
      {current.isLoading?<div className="pos-shift-loading" aria-label="Cargando turno" aria-busy="true"><i/><i/></div>
      :shift?<><div className="pos-shift-copy"><small>TURNO ACTIVO</small><b>{shift.cashRegisterName}<span>· {shift.code}</span></b><p>Cajero: {cashShiftAttribution(shift).name}</p></div><Status tone="green">Listo para cobrar</Status></>
      :<><div><small>TURNO REQUERIDO</small><b>No estás asignado a una caja abierta</b><p>Inicia o únete a un turno antes de registrar cobros o devoluciones.</p></div><Link href={pageRoutes.cash} className="button secondary"><Icon name="register" size={16}/><span>Ir a Caja</span></Link></>}
    </section>

    <section className="panel standardized-management pos-panel">
      <div className="pos-toolbar">
        <label className="pos-search"><Icon name="search" size={17}/><Input value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder="Buscar pedido, cliente o mesa..."/></label>
        <Select aria-label="Filtrar por estado de cobro" value={paymentStatus} onChange={event=>{setPaymentStatus(event.target.value);setPage(1)}}>
          <option value="unpaid">Por cobrar</option>
          <option value="pending">Sin pagos</option>
          <option value="partial">Pago parcial</option>
          <option value="paid">Pagados</option>
          <option value="all">Todos</option>
        </Select>
      </div>

      {orders.isLoading?<POSLoading/>
      :orders.isError?<POSState icon="alert" title="No pudimos cargar los pedidos" text={orders.error.message} retry={()=>orders.refetch()}/>
      :!items.length?<POSState icon="receipt" title={q?"Sin coincidencias":paymentStatus==="paid"?"Aún no hay pedidos pagados":"No hay pedidos pendientes de cobro"} text={q?"Prueba con otro término de búsqueda.":"Los pedidos aparecerán aquí según su estado de cobro."}/>
      :<div className="table-wrap hover-scroll pos-table-wrap"><table className="pos-table">
        <thead><tr><th>PEDIDO</th><th>CANAL</th><th className="pos-money">TOTAL</th><th className="pos-money">PAGADO</th><th className="pos-money">SALDO</th><th>COBRO</th><th className="pos-th-actions">ACCIONES</th></tr></thead>
        <tbody>{items.map((item,index)=>{
          const meta=paymentMeta[item.paymentStatus];
          const subject=item.tableName||item.customerName||item.code;
          return <tr className={index%2?"alternate":""} key={item.id}>
            <td><span className="row-icon pos-row-icon"><Icon name="receipt" size={17}/></span><b>{subject}</b></td>
            <td><span className="pos-channel-cell"><Icon name={channelIcons[item.channel]??"receipt"} size={13}/>{channelLabel[item.channel]??item.channel}</span></td>
            <td className="pos-money"><b>{money(Number(item.total))}</b></td>
            <td className="pos-money">{money(Number(item.paidAmount))}</td>
            <td className="pos-money"><strong className="pos-balance">{money(Number(item.remainingAmount))}</strong></td>
            <td><Status tone={meta.tone}>{meta.label}</Status></td>
            <td>{actionsFor(item)}</td>
          </tr>;
        })}</tbody>
      </table></div>}

      {!orders.isLoading&&!orders.isError&&items.length>0&&<div className="management-cards pos-order-cards">{items.map(item=>{
        const meta=paymentMeta[item.paymentStatus];
        const subject=item.tableName||item.customerName||item.code;
        return <article key={item.id}>
          <header><span className="row-icon pos-row-icon"><Icon name="receipt" size={17}/></span><div><b>{subject}</b></div><Status tone={meta.tone}>{meta.label}</Status></header>
          <div className="pos-card-channel"><Icon name={channelIcons[item.channel]??"receipt"} size={14}/><span>{channelLabel[item.channel]??item.channel}</span></div>
          <dl><div><dt>Total</dt><dd>{money(Number(item.total))}</dd></div><div><dt>Pagado</dt><dd>{money(Number(item.paidAmount))}</dd></div><div className="remaining"><dt>Saldo</dt><dd>{money(Number(item.remainingAmount))}</dd></div></dl>
          <footer>{actionsFor(item)}</footer>
        </article>;
      })}</div>}

      {!orders.isLoading&&!orders.isError&&<Pagination page={page} size={size} total={orders.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
    </section>

    {paymentTarget&&shift&&<PaymentDialog order={paymentTarget} shiftName={shift.cashRegisterName+" · "+shift.code} busy={pay.isPending} formatMoney={money} close={()=>setPaymentTarget(null)} save={draft=>pay.mutate({orderId:paymentTarget.id,draft})}/>}
    {refundTarget&&shift&&<RefundDialog payment={refundTarget} busy={refund.isPending} formatMoney={money} close={()=>setRefundTarget(null)} save={draft=>refund.mutate({paymentId:refundTarget.id,draft})}/>}
    {detailId&&<POSDetailDialog
      loading={detail.isLoading}
      error={detail.error?.message}
      data={detail.data}
      canManage={canManage}
      hasShift={Boolean(shift)}
      formatMoney={money}
      formatDateTime={dateTime}
      close={()=>setDetailId(null)}
      refund={setRefundTarget}
    />}
  </div>;
}

function POSDetailDialog({loading,error,data,canManage,hasShift,formatMoney,formatDateTime,close,refund}:{loading:boolean;error?:string;data?:POSOrderDetail;canManage:boolean;hasShift:boolean;formatMoney:(value:number)=>string;formatDateTime:(value:string)=>string;close:()=>void;refund:(payment:Payment)=>void}){
  const closed=Boolean(data&&["entregado","cancelado"].includes(data.order.status));
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal pos-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-detail-title">
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div className="pos-modal-heading"><small>DETALLE DE COBRO</small><h2 id="pos-detail-title">{data?.order.code??"Pedido"}</h2>{data&&<p>{data.order.tableName||data.order.customerName||channelLabel[data.order.channel]||"Pedido del local"}</p>}</div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
    {loading?<POSDetailLoading/>
    :error?<POSState icon="alert" title="No pudimos cargar el cobro" text={error}/>
    :data&&<div className="pos-detail-body">
      <section className="pos-detail-summary">
        <div><small>TOTAL</small><b>{formatMoney(Number(data.order.total))}</b></div>
        <div><small>PAGADO NETO</small><b>{formatMoney(Number(data.paidAmount))}</b></div>
        <div className="remaining"><small>SALDO</small><strong>{formatMoney(Number(data.remainingAmount))}</strong></div>
        <Status tone={paymentMeta[data.paymentStatus].tone}>{paymentMeta[data.paymentStatus].label}</Status>
      </section>

      <section className="pos-detail-section">
        <header><div><small>CONSUMO</small><h3>Productos del pedido</h3></div><span>{data.order.items?.length??0} líneas</span></header>
        <div className="pos-order-lines">{(data.order.items??[]).map(item=><div key={item.id}><b>{Number(item.qty)}×</b><span>{item.name}{item.note&&<small><Icon name="edit" size={11}/>{item.note}</small>}</span><strong>{formatMoney(Number(item.qty)*Number(item.unitPrice))}</strong></div>)}</div>
      </section>

      <section className="pos-detail-section">
        <header><div><small>PAGOS</small><h3>Historial del pedido</h3></div><span>{data.payments.length}</span></header>
        {data.payments.length?<div className="pos-payment-list">{data.payments.map(payment=>{
          const net=Number(payment.netAmount);
          return <article key={payment.id}><span className={"pos-payment-method pm-"+payment.method}><Icon name={paymentMethodMeta[payment.method]?.icon??"wallet"} size={15}/></span><div><b>{paymentMethodMeta[payment.method]?.label??"Otro"}</b><small>{payment.cashRegisterName} · {formatDateTime(payment.createdAt)} · {payment.createdByName}</small>{payment.reference&&<em>{payment.reference}</em>}</div><span className="pos-payment-values"><b>{formatMoney(Number(payment.amount))}</b>{Number(payment.refundedAmount)>0&&<small>Devuelto {formatMoney(Number(payment.refundedAmount))}</small>}</span>{canManage&&net>0&&!closed&&<Button kind="ghost" className="pos-refund-action" icon="undo" disabled={!hasShift} onClick={()=>refund(payment)}>Devolver</Button>}</article>})}</div>
        :<div className="pos-payments-empty">Aún no se registraron pagos para este pedido.</div>}
      </section>
    </div>}
  </section></div>;
}

function POSLoading(){return <>
  <div className="table-wrap pos-table-wrap pos-table-skeleton" aria-label="Cargando pedidos" aria-busy="true"><table className="pos-table"><thead><tr><th>PEDIDO</th><th>CANAL</th><th>TOTAL</th><th>PAGADO</th><th>SALDO</th><th>COBRO</th><th>ACCIONES</th></tr></thead><tbody>{Array.from({length:5},(_,index)=><tr key={index}><td><span className="pos-sk-entity"><i/><span><b/></span></span></td>{Array.from({length:5},(_,cell)=><td key={cell}><i className="pos-sk-line"/></td>)}<td><span className="pos-sk-actions"><i/><i/></span></td></tr>)}</tbody></table></div>
  <div className="management-cards pos-order-cards pos-card-skeleton" aria-hidden="true">{Array.from({length:4},(_,index)=><article key={index}><header><i/><span><b/></span><i/></header><i/><div/><footer><i/><i/></footer></article>)}</div>
</>}

function POSDetailLoading(){return <div className="pos-detail-body pos-detail-loading" aria-label="Cargando detalle del pedido" aria-busy="true">
  <section className="pos-detail-summary pos-detail-summary-skeleton">{Array.from({length:4},(_,index)=><i key={index}/>)}</section>
  {Array.from({length:2},(_,section)=><section className="pos-detail-section pos-detail-section-skeleton" key={section}><header><span><i/><b/></span><i/></header>{Array.from({length:3},(_,row)=><div key={row}><i/><span><b/><small/></span><i/></div>)}</section>)}
</div>}

function POSState({icon,title,text,retry}:{icon:"alert"|"receipt";title:string;text:string;retry?:()=>void}){return <div className="pos-state"><span><Icon name={icon} size={22}/></span><b>{title}</b><p>{text}</p>{retry&&<Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button>}</div>}
