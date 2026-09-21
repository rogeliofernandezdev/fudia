"use client";

import {Button,Input,Label} from "@/components/ui/controls";
import {useEffect,useMemo,useState} from "react";
import {useRouter} from "next/navigation";
import {Icon} from "@/components/icon";
import {operationsFetch,POSOrderDetail} from "@/lib/operations-api";

const money=(value:number|string)=>`S/ ${Number(value||0).toFixed(2)}`;

export default function ReceiptPage(){
  const router=useRouter();
  const[orderId,setOrderId]=useState("");
  const[data,setData]=useState<POSOrderDetail|null>(null);
  const[receiptType,setReceiptType]=useState("boleta");
  const[documentId,setDocumentId]=useState("");
  const[loading,setLoading]=useState(true);
  const[busy,setBusy]=useState(false);
  const[error,setError]=useState("");

  useEffect(()=>{
    const id=new URLSearchParams(window.location.search).get("orderId")??"";
    setOrderId(id);
    if(!id){setError("No se encontró el pedido cobrado.");setLoading(false);return;}
    void operationsFetch<POSOrderDetail>(`pos/orders/${id}`).then(order=>{
      setData(order);
      if(order.paymentStatus!=="paid")setError("El pedido todavía tiene saldo pendiente.");
    }).catch(e=>setError(e instanceof Error?e.message:"No se pudo cargar el comprobante.")).finally(()=>setLoading(false));
  },[]);

  const paid=useMemo(()=>data?.payments.reduce((sum,p)=>sum+Number(p.netAmount||0),0)??0,[data]);
  const cash=useMemo(()=>data?.payments.filter(p=>p.method==="cash").reduce((sum,p)=>sum+Number(p.netAmount||0),0)??0,[data]);
  const methods=useMemo(()=>{
    if(!data)return "";
    const labels=[...new Set(data.payments.map(p=>p.method==="cash"?"Efectivo":p.method==="card"?"Tarjeta":p.method==="transfer"?"Yape/Plin o transferencia":"Otro"))];
    return labels.join(" + ");
  },[data]);

  async function finalize(){
    if(!orderId||!data||data.paymentStatus!=="paid"||busy)return;
    setBusy(true);setError("");
    try{
      await operationsFetch<void>(`pos/orders/${orderId}/complete`,{method:"POST"});
      router.push("/mesas");
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

  if(loading)return <div className="pos-empty"><b>Cargando comprobante…</b><span>Verificando el pago registrado.</span></div>;
  if(!data)return <div className="pos-empty"><Icon name="alert" size={22}/><b>No se pudo abrir el comprobante</b><span>{error}</span></div>;

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
    <Button tone="primary" className="wide" disabled={busy||data.paymentStatus!=="paid"} onClick={()=>void finalize()}>{busy?"Finalizando…":"Finalizar y liberar mesa"}<Icon name="chevron" size={17}/></Button>
  </div></div>;
}
