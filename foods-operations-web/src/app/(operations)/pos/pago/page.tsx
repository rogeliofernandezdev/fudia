"use client";

import {Button,Input,Label} from "@/components/ui/controls";
import {Suspense,useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useRouter,useSearchParams} from "next/navigation";
import {Icon} from "@/components/icon";
import {CashShift,operationsFetch,PaymentMethod,POSOrderDetail} from "@/lib/operations-api";

const money=(value:number|string)=>`S/ ${Number(value||0).toFixed(2)}`;

export default function PaymentPage(){
  return <Suspense fallback={<div className="pos-empty"><b>Cargando cobro…</b><span>Preparando la transacción.</span></div>}><PaymentContent/></Suspense>;
}

function PaymentContent(){
  const router=useRouter();
  const searchParams=useSearchParams();
  const orderId=searchParams.get("orderId")??"";
  const[data,setData]=useState<POSOrderDetail|null>(null);
  const[shift,setShift]=useState<CashShift|null>(null);
  const[methods,setMethods]=useState<PaymentMethod[]>([]);
  const[method,setMethod]=useState("");
  const[received,setReceived]=useState("");
  const[reference,setReference]=useState("");
  const[split,setSplit]=useState<Record<string,string>>({});
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
      operationsFetch<{items:PaymentMethod[]}>("payment-methods?usage=sales"),
    ]).then(([order,current,catalog])=>{
      if(!active)return;
      setData(order);setShift(current.shift);setMethods(catalog.items);
      setMethod(catalog.items[0]?.code??"");
      setReceived(Number(order.remainingAmount).toFixed(2));
    }).catch(e=>{
      if(active)setError(e instanceof Error?e.message:"No se pudo cargar el cobro.");
    }).finally(()=>{if(active)setLoading(false)});
    return()=>{active=false};
  },[orderId]);

  const remaining=Number(data?.remainingAmount??0);
  const subtotal=Number(data?.order.total??0)/1.18;
  const igv=Number(data?.order.total??0)-subtotal;
  const selectedMethod=methods.find(item=>item.code===method);
  const receivedValue=Number(received)||0;
  const change=selectedMethod?.affectsCash?Math.max(0,receivedValue-remaining):0;
  const splitTotal=methods.reduce((sum,item)=>sum+(Number(split[item.code])||0),0);
  const cashSplitMethod=methods.find(item=>item.affectsCash&&(Number(split[item.code])||0)>0);
  const splitCashAmount=cashSplitMethod?Number(split[cashSplitMethod.code])||0:0;
  const splitChange=Math.max(0,(Number(splitReceived)||0)-splitCashAmount);
  const splitBalanced=Math.abs(splitTotal-remaining)<0.005;
  const canConfirm=Boolean(shift)&&remaining>0&&!busy&&Boolean(selectedMethod||method==="split")&&(
    selectedMethod?.affectsCash?receivedValue>=remaining:
    method==="split"?splitBalanced&&(splitCashAmount<=0||(Number(splitReceived)||0)>=splitCashAmount):
    true
  );
  const lines=useMemo(()=>data?.order.items??[],[data]);

  async function confirm(){
    if(!data||!orderId||!canConfirm)return;
    setBusy(true);setError("");
    const payments:{method:string;amount:number;reference:string}[]=[];
    if(method!=="split"&&selectedMethod){
      payments.push({
        method:selectedMethod.code,
        amount:remaining,
        reference:selectedMethod.affectsCash?`Recibido ${receivedValue.toFixed(2)} · Vuelto ${change.toFixed(2)}`:reference.trim(),
      });
    }else{
      for(const item of methods){
        const amount=Number(split[item.code])||0;
        if(amount<=0)continue;
        payments.push({
          method:item.code,
          amount,
          reference:item.affectsCash?`Recibido ${Number(splitReceived).toFixed(2)} · Vuelto ${splitChange.toFixed(2)}`:reference.trim(),
        });
      }
    }
    try{
      const result=await operationsFetch<{remainingAmount:string;paymentStatus:string}>("payments/batch",{method:"POST",body:JSON.stringify({orderId,payments})});
      if(Number(result.remainingAmount)>0.005){
        const fresh=await operationsFetch<POSOrderDetail>(`pos/orders/${orderId}`);
        setData(fresh);setMethod(methods[0]?.code??"");setReceived(fresh.remainingAmount);setSplit({});
        setError(`Pago registrado. Aún queda ${money(result.remainingAmount)} por cobrar.`);
        return;
      }
      router.push(`/pos/comprobante?orderId=${orderId}`);
    }catch(e){setError(e instanceof Error?e.message:"No se pudo registrar el pago.")}
    finally{setBusy(false)}
  }

  if(!orderId)return <div className="flow-page"><header className="flow-header"><Link href="/caja"><Icon name="chevron" size={18}/>Volver a Caja</Link><div><span>COBRO</span><h1>No se puede procesar</h1><p>Selecciona una cuenta real para cobrar.</p></div></header></div>;
  if(loading)return <div className="pos-empty"><b>Cargando cobro…</b><span>Validando pedido, turno y medios de pago.</span></div>;
  if(!data)return <div className="flow-page"><header className="flow-header"><Link href="/caja"><Icon name="chevron" size={18}/>Volver a Caja</Link><div><span>COBRO</span><h1>No se puede procesar</h1><p>{error}</p></div></header></div>;

  return <div className="flow-page"><header className="flow-header"><Link href="/caja"><Icon name="chevron" size={18}/>Volver a Caja</Link><div><span>COBRO DE PEDIDO</span><h1>Procesar pago</h1><p>{data.order.tableName||data.order.customerName||data.order.code} · {data.order.code}</p></div></header>
    {!shift&&<div className="missing-card"><span>Debes tener un turno de caja activo para cobrar.</span><Link href="/turno">Abrir turno</Link></div>}
    {!methods.length&&<div className="missing-card"><span>No hay medios de pago activos para ventas.</span></div>}
    {error&&<div className="missing-card"><span>{error}</span></div>}
    <div className="payment-layout">
      <section className="payment-methods">
        <h2>Selecciona el medio de pago</h2>
        <div className="method-grid">
          {methods.map(item=><Button layout="card" className={method===item.code?"active":""} key={item.code} onClick={()=>setMethod(item.code)} disabled={busy}>
            <Icon name={item.affectsCash?"cash":"card"} size={23}/><span><b>{item.name}</b><small>{item.description}</small></span>{method===item.code&&<Icon name="check" size={17}/>}
          </Button>)}
          {methods.length>1&&<Button layout="card" className={method==="split"?"active":""} onClick={()=>setMethod("split")} disabled={busy}>
            <Icon name="plus" size={23}/><span><b>Pago dividido</b><small>Varios medios</small></span>{method==="split"&&<Icon name="check" size={17}/>}
          </Button>}
        </div>
        {selectedMethod?.affectsCash&&<>
          <Label className="amount-field"><span>Efectivo recibido</span><div><b>S/</b><Input inputMode="decimal" aria-label="Monto recibido" value={received} onChange={e=>setReceived(e.target.value)}/></div></Label>
          <div className="quick-amounts"><Button onClick={()=>setReceived(remaining.toFixed(2))}>{money(remaining)} exacto</Button><Button onClick={()=>setReceived("50.00")}>S/ 50.00</Button><Button onClick={()=>setReceived("100.00")}>S/ 100.00</Button></div>
          {receivedValue<remaining&&<div className="missing-card"><span>Falta recibir</span><strong>{money(remaining-receivedValue)}</strong></div>}
          <div className="change-card"><span>Vuelto</span><strong>{money(change)}</strong></div>
        </>}
        {selectedMethod&&!selectedMethod.affectsCash&&<Label className="amount-field"><span>Referencia / operación</span><Input value={reference} maxLength={120} onChange={e=>setReference(e.target.value)} placeholder="Opcional"/></Label>}
        {method==="split"&&<div className="split-payment-fields">
          {methods.map(item=><Label key={item.code}><span>{item.name}</span><Input inputMode="decimal" value={split[item.code]??""} onChange={e=>setSplit(v=>({...v,[item.code]:e.target.value}))} placeholder="0.00"/></Label>)}
          {splitCashAmount>0&&<Label><span>Efectivo recibido</span><Input inputMode="decimal" value={splitReceived} onChange={e=>setSplitReceived(e.target.value)} placeholder={splitCashAmount.toFixed(2)}/></Label>}
          <Label><span>Referencia para medios no efectivos</span><Input value={reference} maxLength={120} onChange={e=>setReference(e.target.value)} placeholder="Opcional"/></Label>
          <div className={splitBalanced?"change-card":"missing-card"}><span>{splitBalanced?"Distribución completa":"Total distribuido"}</span><strong>{money(splitTotal)} / {money(remaining)}</strong></div>
          {splitCashAmount>0&&<div className="change-card"><span>Vuelto efectivo</span><strong>{money(splitChange)}</strong></div>}
        </div>}
      </section>
      <aside className="payment-summary"><span>RESUMEN DEL PEDIDO</span><h2>{data.order.tableName||data.order.code}</h2>
        <div className="summary-lines">{lines.map(item=><p key={item.id}><span>{Number(item.qty)}× {item.name}</span><b>{money(Number(item.qty)*Number(item.unitPrice))}</b></p>)}</div>
        <dl><div><dt>Total pedido</dt><dd>{money(data.order.total)}</dd></div><div><dt>Pagado</dt><dd>{money(data.paidAmount)}</dd></div><div><dt>Saldo a cobrar</dt><dd>{money(data.remainingAmount)}</dd></div><div><dt>IGV referencial incluido</dt><dd>{money(igv)}</dd></div></dl>
        <Button tone="primary" className="wide" disabled={!canConfirm} onClick={()=>void confirm()}><Icon name="check"/>{busy?"Registrando pago…":"Confirmar pago"}</Button>
        <small>Los medios disponibles se cargan desde la configuración de la empresa. Solo los medios marcados como efectivo impactan Caja.</small>
      </aside>
    </div>
  </div>;
}
