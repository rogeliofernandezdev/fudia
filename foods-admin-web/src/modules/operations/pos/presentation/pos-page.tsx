"use client";
import "./pos.css";
import Link from "next/link";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,RowActionButton,Select,Status} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {getCurrentCashShift} from "../../cash/infrastructure/cash-api";
import type {Payment,POSOrderDetail,POSOrderSummary} from "../domain/types";
import {createPayment,getPOSOrder,listPOSOrders,refundPayment} from "../infrastructure/pos-api";
import {PaymentDialog,RefundDialog} from "./pos-dialogs";

const paymentMeta={
  pending:{label:"Sin pagos",tone:"gray" as const},
  partial:{label:"Pago parcial",tone:"orange" as const},
  paid:{label:"Pagado",tone:"green" as const},
};
const channelLabel:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};

export function POSPage({initialOrderId=""}:{initialOrderId?:string}){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const settings=useSettings();
  const canManage=can("cash.manage");
  const[q,setQ]=useState("");
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
    queryKey:["pos-orders",q,paymentStatus,page,size],
    queryFn:()=>listPOSOrders({q,paymentStatus,page,pageSize:size}),
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
  function summaryFromDetail(data:POSOrderDetail):POSOrderSummary{
    return{
      id:data.order.id,code:data.order.code,channel:data.order.channel,status:data.order.status,
      customerName:data.order.customerName,tableName:data.order.tableName,total:data.order.total,
      paidAmount:data.paidAmount,remainingAmount:data.remainingAmount,paymentStatus:data.paymentStatus,createdAt:data.order.createdAt,
    };
  }

  return <div className="pos-page">
    <PageHeader eyebrow="OPERACIÓN" title="Punto de venta" description="Cobra pedidos existentes y mantiene Caja sincronizada con pagos y devoluciones."/>

    <section className={"pos-shift-banner "+(shift?"active":"missing")}>
      <span className="pos-shift-icon"><Icon name={shift?"sales":"alert"} size={19}/></span>
      {current.isLoading?<div><small>TURNO DE CAJA</small><b>Cargando turno…</b></div>
      :shift?<><div><small>TURNO ACTIVO</small><b>{shift.cashRegisterName} · {shift.code}</b><p>Operado por {shift.openedByName}</p></div><Status tone="green">Listo para cobrar</Status></>
      :<><div><small>TURNO REQUERIDO</small><b>No estás asignado a una caja abierta</b><p>Inicia o únete a un turno antes de registrar cobros o devoluciones.</p></div><Link href="/caja" className="button secondary"><Icon name="sales" size={16}/><span>Ir a Caja</span></Link></>}
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
        <p><Icon name="receipt" size={14}/>Los cobros en efectivo actualizan automáticamente el turno activo.</p>
      </div>

      {orders.isLoading?<POSLoading/>
      :orders.isError?<POSState icon="alert" title="No pudimos cargar los pedidos" text={orders.error.message} retry={()=>orders.refetch()}/>
      :!items.length?<POSState icon="sales" title={q?"Sin coincidencias":paymentStatus==="paid"?"Aún no hay pedidos pagados":"No hay pedidos pendientes de cobro"} text={q?"Prueba con otro término de búsqueda.":"Los pedidos aparecerán aquí según su estado de cobro."}/>
      :<div className="table-wrap hover-scroll pos-table-wrap"><table className="pos-table">
        <thead><tr><th>PEDIDO</th><th>CANAL</th><th>TOTAL</th><th>PAGADO</th><th>SALDO</th><th>COBRO</th><th>ACCIONES</th></tr></thead>
        <tbody>{items.map((item,index)=>{
          const meta=paymentMeta[item.paymentStatus];
          const subject=item.tableName||item.customerName||"Pedido";
          return <tr className={index%2?"alternate":""} key={item.id}>
            <td><span className="row-icon"><Icon name="receipt" size={17}/></span><b>{subject}</b><small>{item.code} · {dateTime(item.createdAt)}</small></td>
            <td>{channelLabel[item.channel]??item.channel}</td>
            <td><b>{money(Number(item.total))}</b></td>
            <td>{money(Number(item.paidAmount))}</td>
            <td><strong className="pos-balance">{money(Number(item.remainingAmount))}</strong></td>
            <td><Status tone={meta.tone}>{meta.label}</Status></td>
            <td><div className="pos-actions"><RowActionButton action="view" onClick={()=>setDetailId(item.id)}/>{canManage&&item.paymentStatus!=="paid"&&<Button className="pos-pay-row" icon="sales" disabled={!shift} onClick={()=>setPaymentTarget(item)}>Cobrar</Button>}</div></td>
          </tr>;
        })}</tbody>
      </table></div>}

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
      pay={data=>setPaymentTarget(summaryFromDetail(data))}
      refund={setRefundTarget}
    />}
  </div>;
}

function POSDetailDialog({loading,error,data,canManage,hasShift,formatMoney,formatDateTime,close,pay,refund}:{loading:boolean;error?:string;data?:POSOrderDetail;canManage:boolean;hasShift:boolean;formatMoney:(value:number)=>string;formatDateTime:(value:string)=>string;close:()=>void;pay:(data:POSOrderDetail)=>void;refund:(payment:Payment)=>void}){
  const closed=Boolean(data&&["entregado","cancelado"].includes(data.order.status));
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal pos-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-detail-title">
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>DETALLE DE COBRO</small><h2 id="pos-detail-title">{data?.order.code??"Pedido"}</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
    {loading?<div className="pos-detail-loading">{Array.from({length:5},(_,index)=><i key={index}/>)}</div>
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
        <div className="pos-order-lines">{(data.order.items??[]).map(item=><div key={item.id}><b>{Number(item.qty)}×</b><span>{item.name}{item.note&&<small>{item.note}</small>}</span><strong>{formatMoney(Number(item.qty)*Number(item.unitPrice))}</strong></div>)}</div>
      </section>

      <section className="pos-detail-section">
        <header><div><small>PAGOS</small><h3>Historial del pedido</h3></div><span>{data.payments.length}</span></header>
        {data.payments.length?<div className="pos-payment-list">{data.payments.map(payment=>{
          const net=Number(payment.netAmount);
          return <article key={payment.id}><span className="pos-payment-method"><Icon name={payment.method==="cash"?"sales":payment.method==="card"?"receipt":"share"} size={15}/></span><div><b>{payment.method==="cash"?"Efectivo":payment.method==="card"?"Tarjeta":payment.method==="transfer"?"Transferencia":"Otro"}</b><small>{payment.cashRegisterName} · {formatDateTime(payment.createdAt)} · {payment.createdByName}</small>{payment.reference&&<em>{payment.reference}</em>}</div><span className="pos-payment-values"><b>{formatMoney(Number(payment.amount))}</b>{Number(payment.refundedAmount)>0&&<small>Devuelto {formatMoney(Number(payment.refundedAmount))}</small>}</span>{canManage&&net>0&&!closed&&<Button kind="ghost" className="pos-refund-action" disabled={!hasShift} onClick={()=>refund(payment)}>Devolver</Button>}</article>})}</div>
        :<div className="pos-payments-empty">Aún no se registraron pagos para este pedido.</div>}
      </section>
    </div>}
    {data&&canManage&&Number(data.remainingAmount)>0&&data.order.status!=="cancelado"&&<footer className="pos-detail-footer"><Button icon="sales" disabled={!hasShift} onClick={()=>pay(data)}>Cobrar saldo {formatMoney(Number(data.remainingAmount))}</Button></footer>}
  </section></div>;
}

function POSLoading(){return <div className="pos-loading">{Array.from({length:6},(_,index)=><i key={index}/>)}</div>}

function POSState({icon,title,text,retry}:{icon:"alert"|"sales";title:string;text:string;retry?:()=>void}){return <div className="pos-state"><span><Icon name={icon} size={22}/></span><b>{title}</b><p>{text}</p>{retry&&<Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button>}</div>}
