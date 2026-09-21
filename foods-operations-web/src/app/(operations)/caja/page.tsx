"use client";

import {useCallback,useEffect,useState} from "react";
import {ActionLink,Button,Input,Label} from "@/components/ui/controls";
import {Icon} from "@/components/icon";
import {PageHeading} from "@/components/app-shell";
import {CashShift,operationsFetch,POSOrderSummary} from "@/lib/operations-api";

type OrdersResponse={items:POSOrderSummary[];total:number};

const money=(value:string|number)=>`S/ ${Number(value||0).toFixed(2)}`;

export default function CashPage(){
  const[shift,setShift]=useState<CashShift|null>(null);
  const[pending,setPending]=useState<POSOrderSummary[]>([]);
  const[loading,setLoading]=useState(true);
  const[error,setError]=useState("");
  const[closing,setClosing]=useState(false);
  const[counted,setCounted]=useState("");
  const[closeError,setCloseError]=useState("");

  const load=useCallback(async()=>{
    setLoading(true);setError("");
    try{
      const[current,orders]=await Promise.all([
        operationsFetch<{shift:CashShift|null}>("cash-shifts/current"),
        operationsFetch<OrdersResponse>("pos/orders?paymentStatus=unpaid&page=1&pageSize=20"),
      ]);
      setShift(current.shift);
      setPending(orders.items);
    }catch(e){setError(e instanceof Error?e.message:"No se pudo cargar Caja.");}
    finally{setLoading(false);}
  },[]);

  useEffect(()=>{void load();},[load]);

  async function closeShift(){
    if(!shift||closing)return;
    const value=Number(counted);
    if(!Number.isFinite(value)||value<0){setCloseError("Ingresa el efectivo contado.");return;}
    setClosing(true);setCloseError("");
    try{
      await operationsFetch<void>(`cash-shifts/${shift.id}/close`,{method:"POST",body:JSON.stringify({countedAmount:value,note:"Cierre desde Operaciones"})});
      setCounted("");
      await load();
    }catch(e){setCloseError(e instanceof Error?e.message:"No se pudo cerrar la caja.");}
    finally{setClosing(false);}
  }

  if(loading)return <div className="pos-empty"><b>Cargando caja…</b><span>Consultando turno y cuentas pendientes.</span></div>;

  return <>
    <PageHeading eyebrow="CAJA PRINCIPAL" title="Caja y cobros" description={shift?`Turno ${shift.code} · ${shift.cashRegisterName}`:"No tienes un turno de caja activo"} action={<ActionLink tone="operational" href="/pos" className="button primary blue"><Icon name="plus"/>Nueva venta</ActionLink>}/>
    {error&&<div className="missing-card"><span>{error}</span><Button onClick={()=>void load()}>Reintentar</Button></div>}
    {!shift&&<section className="panel close-card"><span className="close-icon"><Icon name="cash"/></span><h2>Necesitas abrir un turno</h2><p>Los cobros requieren una caja abierta y asignada a tu usuario.</p><ActionLink tone="primary" href="/turno">Abrir turno</ActionLink></section>}
    {shift&&<>
      <section className="cash-kpis">
        <article><span className="blue"><Icon name="wallet"/></span><div><small>EFECTIVO ESPERADO</small><b>{shift.expectedVisible?money(shift.expectedAmount):"Cierre ciego"}</b><em>{shift.cashRegisterName} · {shift.code}</em></div></article>
        <article><span className="green"><Icon name="cash"/></span><div><small>INGRESOS EN EFECTIVO</small><b>{shift.expectedVisible?money(shift.incomeAmount):"Oculto"}</b><em>Incluye ventas y otros ingresos</em></div></article>
        <article><span className="violet"><Icon name="card"/></span><div><small>CUENTAS POR COBRAR</small><b>{pending.length}</b><em>Pedidos con saldo pendiente</em></div></article>
      </section>
      <section className="dashboard-grid cash-grid">
        <article className="panel">
          <header><div><span className="section-kicker">PENDIENTES</span><h2>Cuentas por cobrar</h2></div><b className="count">{pending.length}</b></header>
          <div className="pay-list">{pending.map(order=><div key={order.id}><span><b>{order.tableName||order.customerName||order.code}</b><small>{order.code} · {order.paymentStatus==="partial"?"Pago parcial":"Sin pagos"}</small></span><strong>{money(order.remainingAmount)}</strong><ActionLink tone="operational" href={`/pos/pago?orderId=${order.id}`} className="button primary blue">Cobrar</ActionLink></div>)}</div>
          {!pending.length&&<div className="pos-empty"><Icon name="check" size={22}/><b>Sin cuentas pendientes</b><span>Todos los pedidos están cobrados.</span></div>}
        </article>
        <article className="panel close-card">
          <span className="close-icon"><Icon name="cash"/></span>
          <h2>Cierre de turno</h2>
          <p>Registra el efectivo contado. El backend calculará y conservará la diferencia del arqueo.</p>
          <dl>
            <div><dt>Fondo inicial</dt><dd>{shift.expectedVisible?money(shift.openingAmount):"Oculto"}</dd></div>
            <div><dt>Ingresos</dt><dd>{shift.expectedVisible?money(shift.incomeAmount):"Oculto"}</dd></div>
            <div><dt>Egresos</dt><dd>{shift.expectedVisible?`-${money(shift.expenseAmount)}`:"Oculto"}</dd></div>
            <div className="expected"><dt>Efectivo esperado</dt><dd>{shift.expectedVisible?money(shift.expectedAmount):"Cierre ciego"}</dd></div>
          </dl>
          <Label><span>Efectivo contado</span><Input inputMode="decimal" value={counted} onChange={e=>setCounted(e.target.value)} placeholder="0.00"/></Label>
          {closeError&&<small className="wizard-field-error">{closeError}</small>}
          <Button className="button outline wide" disabled={closing} onClick={()=>void closeShift()}>{closing?"Cerrando caja…":"Cerrar caja y guardar arqueo"}</Button>
        </article>
      </section>
    </>}
  </>;
}
