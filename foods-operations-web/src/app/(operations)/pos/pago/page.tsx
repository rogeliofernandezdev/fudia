"use client";
import { ActionLink } from "@/components/ui/controls";


import { Button, Input, Label } from "@/components/ui/controls";


import { useState } from "react";
import Link from "next/link";
import { Icon } from "@/components/icon";

const total = 44;
const igv = 6.71;
const subtotal = total - igv;

const methods = [
  { id: "cash", label: "Efectivo", detail: "Pago en caja", icon: "cash" as const },
  { id: "card", label: "Tarjeta", detail: "POS físico", icon: "card" as const },
  { id: "wallet", label: "Yape o Plin", detail: "Billetera digital", icon: "wallet" as const },
  { id: "split", label: "Pago dividido", detail: "Varios medios", icon: "plus" as const },
];

export default function PaymentPage() {
  const [method, setMethod] = useState("cash");
  const [received, setReceived] = useState("50.00");
  const receivedValue = parseFloat(received) || 0;
  const change = method === "cash" ? Math.max(0, receivedValue - total) : 0;
  const missing = method === "cash" ? Math.max(0, total - receivedValue) : 0;
  const canConfirm = method !== "cash" || receivedValue >= total;

  return <div className="flow-page"><header className="flow-header"><Link href="/pos"><Icon name="chevron" size={18}/>Volver al pedido</Link><div><span>COBRO DE MESA</span><h1>Procesar pago</h1><p>Mesa 04 · Pedido #0128</p></div></header>
    <div className="payment-layout">
      <section className="payment-methods">
        <h2>Selecciona el medio de pago</h2>
        <div className="method-grid">{methods.map(m => <Button layout="card" className={method === m.id ? "active" : ""} key={m.id} onClick={() => setMethod(m.id)}>
          <Icon name={m.icon} size={23}/><span><b>{m.label}</b><small>{m.detail}</small></span>{method === m.id && <Icon name="check" size={17}/>}
        </Button>)}</div>
        {method === "cash" && <>
          <Label className="amount-field"><span>Monto recibido</span><div><b>S/</b><Input inputMode="decimal" aria-label="Monto recibido" value={received} onChange={e => setReceived(e.target.value)}/></div></Label>
          <div className="quick-amounts"><Button onClick={() => setReceived(total.toFixed(2))}>S/ {total.toFixed(2)} exacto</Button><Button onClick={() => setReceived("50.00")}>S/ 50.00</Button><Button onClick={() => setReceived("100.00")}>S/ 100.00</Button></div>
          {missing > 0 && <div className="missing-card"><span>Falta recibir</span><strong>S/ {missing.toFixed(2)}</strong></div>}
          <div className="change-card"><span>Vuelto</span><strong>S/ {change.toFixed(2)}</strong></div>
        </>}
        {method !== "cash" && <div className="no-change-note"><Icon name="check" size={18}/><span><b>Monto exacto</b><small>El total se cobrará por {methods.find(m => m.id === method)?.label}.</small></span></div>}
      </section>
      <aside className="payment-summary"><span>RESUMEN DEL PEDIDO</span><h2>Mesa 04</h2><div className="summary-lines"><p><span>1× Lomo saltado</span><b>S/ 28.00</b></p><p><span>2× Chicha morada</span><b>S/ 16.00</b></p></div><dl><div><dt>Subtotal</dt><dd>S/ {subtotal.toFixed(2)}</dd></div><div><dt>IGV (18%)</dt><dd>S/ {igv.toFixed(2)}</dd></div><div><dt>Total</dt><dd>S/ {total.toFixed(2)}</dd></div></dl>
        {canConfirm ? <ActionLink tone="primary" href="/pos/comprobante"><Icon name="check" size={19}/>Confirmar pago<Icon name="chevron" size={17}/></ActionLink> : <Button tone="primary" className="wide" disabled><Icon name="check"/>Confirmar pago</Button>}
        <small>Esta acción registrará el pago y cerrará la mesa.</small></aside>
    </div></div>;
}
