"use client";

import {Button,Input,Label} from "@/components/ui/controls";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useRouter,useSearchParams} from "next/navigation";
import {Icon} from "@/components/icon";
import {CashShift,operationsFetch,POSOrderDetail} from "@/lib/operations-api";

const methods=[
  {id:"cash",label:"Efectivo",detail:"Pago en caja",icon:"cash" as const},
  {id:"card",label:"Tarjeta",detail:"POS físico",icon:"card" as const},
  {id:"wallet",label:"Yape o Plin",detail:"Billetera digital",icon:"wallet" as const},
  {id:"split",label:"Pago dividido",detail:"Varios medios",icon:"plus" as const},
];
const money=(value:number|string)=>`S/ ${Number(value||0).toFixed(2)}`;

export default function PaymentPage(){
  const router=useRouter();
  const searchParams=useSearchParams();
  const orderId=searchParams.get("orderId")??"";
  const[data,setData]=useState<POSOrderDetail|null>(null);
  const[shift,setShift]=useState<CashShift|null>(null);
  const[method,setMethod]=useState("cash");
  const[received,setReceived]=useState("");
  const[reference,setReference]=useState("");
  const[split,setSplit]=useState({cash:"",card:"",wallet:""});
  const[splitReceived,setSplitReceived]=useState("");
  const[loading,setLoading]=useState(true);
  const[busy,setBusy]=useState(false);
  const[error,setError]=useState("");

  useEffect(()=>{
    if(!orderId)return;
    let active=true;
    void Promise.all([
      operationsFetch<POSOrderDetail>(`pos/orders/${orderId}`),
      operationsFetch<{shift:CashShift|null}>("cash-shifts/current"),
    ]).then(([order,current])=>{
      if(!active)return;
      setData(order);setShift(current.shift);
      setReceived(Number(order.remainingAmount).toFixed(2));
    }).catch(e=>{
      if(active)setError(e instanceof Error?e.message:"No se pudo cargar el cobro.");
    }).finally(()=>{
      if(active)setLoading(false);
    });
    return()=>{active=false;};
  },[orderId]);

  const remaining=Number(data?.remainingAmount??0);
  const subtotal=Number(data?.order.total??0)/1.18;
  const igv=Number(data?.order.total??0)-subtotal;
  const receivedValue=Number(received)||0;
  const change=method==="cash"?Math.max(0,receivedValue-remaining):0;
  const splitCash=Number(split.cash)||0;
  const splitCard=Number(split.card)||0;
  const splitWallet=Number(split.wallet)||0;
  const splitTotal=splitCash+splitCard+splitWallet;
  const splitChange=Math.max(0,(Number(splitReceived)||0)-splitCash);
  const splitBalanced=Math.abs(splitTotal-remaining)<0.005;
  const canConfirm=Boolean(shift)&&remaining>0&&!busy&&(
    method==="cash"?receivedValue>=remaining:
    method==="split"?splitBalanced&&(splitCash<=0||(Number(splitReceived)||0)>=splitCash):
    true
  );

  const lines=useMemo(()=>data?.order.items??[],[data]);

  async function confirm(){
    if(!data||!orderId||!canConfirm)return;
    setBusy(true);setError("");
    const payments:{method:string;amount:number;reference:string}[]=[];
    if(method==="cash")payments.push({method:"cash",amount:remaining,reference:`Recibido ${receivedValue.toFixed(2)} · Vuelto ${change.toFixed(2)}`});
    else if(method==="card")payments.push({method:"card",amount:remaining,reference:reference.trim()});
    else if(method==="wallet")payments.push({method:"transfer",amount:remaining,reference:reference.trim()||"Yape/Plin"});
    else{
      if(splitCash>0)payments.push({method:"cash",amount:splitCash,reference:`Recibido ${Number(splitReceived).toFixed(2)} · Vuelto ${splitChange.toFixed(2)}`});
      if(splitCard>0)payments.push({method:"card",amount:splitCard,reference:reference.trim()});
      if(splitWallet>0)payments.push({method:"transfer",amount:splitWallet,reference:"Yape/Plin"});
    }
    try{
      const result=await operationsFetch<{remainingAmount:string;paymentStatus:string}>("payments/batch",{method:"POST",body:JSON.stringify({orderId,payments})});
      if(Number(result.remainingAmount)>0.005){
        const fresh=await operationsFetch<POSOrderDetail>(`pos/orders/${orderId}`);
        setData(fresh);setMethod("cash");setReceived(fresh.remainingAmount);setSplit({cash:"",card:"",wallet:""});
        setError(`Pago registrado. Aún queda ${money(result.remainingAmount)} por cobrar.`);
        return;
      }
      router.push(`/pos/comprobante?orderId=${orderId}`);
    }catch(e){setError(e instanceof Error?e.message:"No se pudo registrar el pago.");}
    finally{setBusy(false);}
  }

  if(!orderId)return <div className="flow-page"><header className="flow-header"><Link href="/caja"><Icon name="chevron" size={18}/>Volver a Caja</Link><div><span>COBRO</span><h1>No se puede procesar</h1><p>Selecciona una cuenta real para cobrar.</p></div></header></div>;
  if(loading)return <div className="pos-empty"><b>Cargando cobro…</b><span>Validando pedido y turno de caja.</span></div>;
  if(!data)return <div className="flow-page"><header className="flow-header"><Link href="/caja"><Icon name="chevron" size={18}/>Volver a Caja</Link><div><span>COBRO</span><h1>No se puede procesar</h1><p>{error}</p></div></header></div>;

  return <div className="flow-page"><header className="flow-header"><Link href="/caja"><Icon name="chevron" size={18}/>Volver a Caja</Link><div><span>COBRO DE PEDIDO</span><h1>Procesar pago</h1><p>{data.order.tableName||data.order.customerName||data.order.code} · {data.order.code}</p></div></header>
    {!shift&&<div className="missing-card"><span>Debes tener un turno de caja activo para cobrar.</span><Link href="/turno">Abrir turno</Link></div>}
    {error&&<div className="missing-card"><span>{error}</span></div>}
    <div className="payment-layout">
      <section className="payment-methods">
        <h2>Selecciona el medio de pago</h2>
        <div className="method-grid">{methods.map(m=><Button layout="card" className={method===m.id?"active":""} key={m.id} onClick={()=>setMethod(m.id)} disabled={busy}>
          <Icon name={m.icon} size={23}/><span><b>{m.label}</b><small>{m.detail}</small></span>{method===m.id&&<Icon name="check" size={17}/>}
        </Button>)}</div>
        {method==="cash"&&<>
          <Label className="amount-field"><span>Efectivo recibido</span><div><b>S/</b><Input inputMode="decimal" aria-label="Monto recibido" value={received} onChange={e=>setReceived(e.target.value)}/></div></Label>
          <div className="quick-amounts"><Button onClick={()=>setReceived(remaining.toFixed(2))}>{money(remaining)} exacto</Button><Button onClick={()=>setReceived("50.00")}>S/ 50.00</Button><Button onClick={()=>setReceived("100.00")}>S/ 100.00</Button></div>
          {receivedValue<remaining&&<div className="missing-card"><span>Falta recibir</span><strong>{money(remaining-receivedValue)}</strong></div>}
          <div className="change-card"><span>Vuelto</span><strong>{money(change)}</strong></div>
        </>}
        {(method==="card"||method==="wallet")&&<Label className="amount-field"><span>{method==="card"?"Referencia / voucher":"Código Yape o Plin"}</span><Input value={reference} maxLength={120} onChange={e=>setReference(e.target.value)} placeholder="Opcional"/></Label>}
        {method==="split"&&<div className="split-payment-fields">
          <Label><span>Efectivo</span><Input inputMode="decimal" value={split.cash} onChange={e=>setSplit(v=>({...v,cash:e.target.value}))} placeholder="0.00"/></Label>
          {splitCash>0&&<Label><span>Efectivo recibido</span><Input inputMode="decimal" value={splitReceived} onChange={e=>setSplitReceived(e.target.value)} placeholder={splitCash.toFixed(2)}/></Label>}
          <Label><span>Tarjeta</span><Input inputMode="decimal" value={split.card} onChange={e=>setSplit(v=>({...v,card:e.target.value}))} placeholder="0.00"/></Label>
          <Label><span>Yape / Plin</span><Input inputMode="decimal" value={split.wallet} onChange={e=>setSplit(v=>({...v,wallet:e.target.value}))} placeholder="0.00"/></Label>
          <div className={splitBalanced?"change-card":"missing-card"}><span>{splitBalanced?"Distribución completa":"Total distribuido"}</span><strong>{money(splitTotal)} / {money(remaining)}</strong></div>
          {splitCash>0&&<div className="change-card"><span>Vuelto efectivo</span><strong>{money(splitChange)}</strong></div>}
        </div>}
      </section>
      <aside className="payment-summary"><span>RESUMEN DEL PEDIDO</span><h2>{data.order.tableName||data.order.code}</h2>
        <div className="summary-lines">{lines.map(item=><p key={item.id}><span>{Number(item.qty)}× {item.name}</span><b>{money(Number(item.qty)*Number(item.unitPrice))}</b></p>)}</div>
        <dl><div><dt>Total pedido</dt><dd>{money(data.order.total)}</dd></div><div><dt>Pagado</dt><dd>{money(data.paidAmount)}</dd></div><div><dt>Saldo a cobrar</dt><dd>{money(data.remainingAmount)}</dd></div><div><dt>IGV referencial incluido</dt><dd>{money(igv)}</dd></div></dl>
        <Button tone="primary" className="wide" disabled={!canConfirm} onClick={()=>void confirm()}><Icon name="check"/>{busy?"Registrando pago…":"Confirmar pago"}</Button>
        <small>El cobro se registra en el turno activo. En efectivo, solo el importe de la deuda impacta Caja; el excedente se calcula como vuelto.</small>
      </aside>
    </div>
  </div>;
}
