"use client";

import {Button,Input,Label} from "@/components/ui/controls";
import {Suspense,useEffect,useMemo,useState} from "react";
import {useRouter,useSearchParams} from "next/navigation";
import {Icon} from "@/components/icon";
import {operationsFetch,POSOrderDetail} from "@/lib/operations-api";

const money=(value:number|string)=>`S/ ${Number(value||0).toFixed(2)}`;

export default function ReceiptPage(){
  return <Suspense fallback={<div className="pos-empty"><b>Cargando comprobante…</b><span>Preparando el detalle.</span></div>}><ReceiptContent/></Suspense>;
}

function ReceiptContent(){
  const router=useRouter();
  const searchParams=useSearchParams();
  const orderId=searchParams.get("orderId")??"";
  const[data,setData]=useState<POSOrderDetail|null>(null);
  const[receiptType,setReceiptType]=useState("boleta");
  const[documentId,setDocumentId]=useState("");
  const[loading,setLoading]=useState(true);
  const[busy,setBusy]=useState(false);
  const[error,setError]=useState("");

  useEffect(()=>{
    if(!orderId)return;
    let active=true;
    void operationsFetch<POSOrderDetail>(`pos/orders/${orderId}`).then(order=>{
      if(!active)return;
      setData(order);
      if(order.paymentStatus!=="paid")setError("El pedido todavía tiene saldo pendiente.");
    }).catch(e=>{
      if(active)setError(e instanceof Error?e.message:"No se pudo cargar el comprobante.");
    }).finally(()=>{
      if(active)setLoading(false);
    });
    return()=>{active=false;};
  },[orderId]);

  const paid=useMemo(()=>data?.payments.reduce((sum,p)=>sum+Number(p.netAmount||0),0)??0,[data]);
  const cash=useMemo(()=>data?.payments.filter(p=>p.method==="cash").reduce((sum,p)=>sum+Number(p.netAmount||0),0)??0,[data]);
  const methods=useMemo(()=>{
    if(!data)return "";
    const labels=[...new Set(data.payments.map(p=>p.method==="cash"?"Efectivo":p.method==="card"?"Tarjeta":p.method==="transfer"?"Yape/Plin o transferencia":"Otro"))];
    return labels.join(" + ");
  },[data]);
  const readyToComplete=data?.order.status==="listo"||data?.order.status==="en_camino";

  async function finalize(){
    if(!orderId||!data||data.paymentStatus!=="paid"||busy)return;
    if(!readyToComplete){
      router.push("/pedidos");
      return;
    }
    setBusy(true);setError("");
    try{
      await operationsFetch<void>(`pos/orders/${orderId}/complete`,{method:"POST"});
      router.push(data.order.tableName?"/mesas":"/pedidos");
      router.refresh();
    }catch(e){setError(e instanceof Error?e.message:"No se pudo finalizar el pedido.");}
    finally{setBusy(false);}
  }

  async function share(){
    if(!data)return;
    const text=`Pedido ${data.order.code} · Total ${money(data.order.total)} · Pagado ${money(paid)} · ${methods}`;
    try{
      if(navigator.share)await navigator.share({title:`Comprobante ${data.order.code}`,text});
      else{await navigator.clipboard.writeText(text);setError("Resumen copiado al portapapeles.");}
    }catch{/* El usuario puede cancelar el diálogo nativo. */}
  }

  if(!orderId)return <div className="pos-empty"><Icon name="receipt" size={22}/><b>No se pudo abrir el comprobante</b><span>No se encontró el pedido cobrado.</span></div>;
  if(loading)return <div className="pos-empty"><b>Cargando comprobante…</b><span>Verificando el pago registrado.</span></div>;
  if(!data)return <div className="pos-empty"><Icon name="receipt" size={22}/><b>No se pudo abrir el comprobante</b><span>{error}</span></div>;

  return <div className="success-page"><div className="success-card">
    <span className="success-icon"><Icon name="check" size={30}/></span>
    <small>{data.paymentStatus==="paid"?"PAGO COMPLETADO":"PAGO PENDIENTE"}</small>
    <h1>{money(data.order.total)}</h1>
    <p>{data.paymentStatus==="paid"?`El pago de ${data.order.tableName||data.order.code} fue registrado correctamente.`:`Aún queda ${money(data.remainingAmount)} por cobrar.`}</p>
    {error&&<div className="missing-card"><span>{error}</span></div>}
    <div className="payment-receipt-summary">
      <span><small>Medio</small><b><Icon name="cash" size={14}/>{methods||"—"}</b></span>
      <span><small>Pagado neto</small><b>{money(paid)}</b></span>
      <span><small>Efectivo aplicado</small><b>{money(cash)}</b></span>
      <span><small>Correlativo</small><b>{data.order.code}</b></span>
    </div>
    <div className="receipt-select">
      <h2>Datos de comprobante</h2>
      <div>
        <Button layout="card" className={receiptType==="boleta"?"active":""} onClick={()=>setReceiptType("boleta")}><Icon name="receipt"/><span><b>Boleta</b><small>Datos para emisión</small></span>{receiptType==="boleta"&&<Icon name="check" size={16}/>}</Button>
        <Button layout="card" className={receiptType==="factura"?"active":""} onClick={()=>setReceiptType("factura")}><Icon name="receipt"/><span><b>Factura</b><small>Requiere RUC</small></span>{receiptType==="factura"&&<Icon name="check" size={16}/>}</Button>
      </div>
      <Label><span>{receiptType==="boleta"?"Documento del cliente":"RUC de la empresa"}</span><Input inputMode="numeric" maxLength={receiptType==="factura"?11:8} placeholder={receiptType==="boleta"?"DNI (opcional)":"20XXXXXXXXX"} value={documentId} onChange={e=>setDocumentId(e.target.value.replace(/\D/g,""))}/></Label>
      <small>Estos datos preparan el ticket operativo. La emisión fiscal electrónica requiere el módulo fiscal correspondiente.</small>
    </div>
    <div className="receipt-actions"><Button onClick={()=>window.print()}><Icon name="printer" size={18}/>Imprimir ticket</Button><Button onClick={()=>void share()}><Icon name="whatsapp" size={18}/>Compartir digital</Button></div>
    {!readyToComplete&&data.paymentStatus==="paid"&&<div className="no-change-note"><Icon name="clock" size={18}/><span><b>Pago cerrado</b><small>La comanda sigue en cocina y no se marcará como entregada antes de estar lista.</small></span></div>}
    <Button tone="primary" className="wide" disabled={busy||data.paymentStatus!=="paid"} onClick={()=>void finalize()}>{busy?"Finalizando…":readyToComplete?(data.order.tableName?"Finalizar y liberar mesa":"Finalizar pedido"):"Cerrar comprobante"}<Icon name="chevron" size={17}/></Button>
  </div></div>;
}
