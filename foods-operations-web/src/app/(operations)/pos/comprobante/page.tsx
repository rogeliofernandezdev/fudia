"use client";
import { ActionLink } from "@/components/ui/controls";


import { Button, Input, Label } from "@/components/ui/controls";


import { useState } from "react";
import { Icon } from "@/components/icon";

export default function ReceiptPage() {
  const [receiptType, setReceiptType] = useState("boleta");
  const [documentId, setDocumentId] = useState("");

  return <div className="success-page"><div className="success-card">
    <span className="success-icon"><Icon name="check" size={30}/></span>
    <small>PAGO COMPLETADO</small>
    <h1>S/ 44.00</h1>
    <p>El pago de la Mesa 04 fue registrado correctamente.</p>
    <div className="payment-receipt-summary">
      <span><small>Medio</small><b><Icon name="cash" size={14}/>Efectivo</b></span>
      <span><small>Recibido</small><b>S/ 50.00</b></span>
      <span><small>Vuelto</small><b>S/ 6.00</b></span>
      <span><small>Correlativo</small><b>#0128</b></span>
    </div>
    <div className="receipt-select">
      <h2>Emitir comprobante</h2>
      <div>
        <Button layout="card" className={receiptType === "boleta" ? "active" : ""} onClick={() => setReceiptType("boleta")}><Icon name="receipt"/><span><b>Boleta</b><small>Consumidor final</small></span>{receiptType === "boleta" && <Icon name="check" size={16}/>}</Button>
        <Button layout="card" className={receiptType === "factura" ? "active" : ""} onClick={() => setReceiptType("factura")}><Icon name="receipt"/><span><b>Factura</b><small>Requiere RUC</small></span>{receiptType === "factura" && <Icon name="check" size={16}/>}</Button>
      </div>
      <Label><span>{receiptType === "boleta" ? "Documento del cliente" : "RUC de la empresa"}</span><Input placeholder={receiptType === "boleta" ? "DNI (opcional)" : "20XXXXXXXXX"} value={documentId} onChange={e => setDocumentId(e.target.value)}/></Label>
    </div>
    <div className="receipt-actions"><Button><Icon name="printer" size={18}/>Imprimir ticket</Button><Button><Icon name="whatsapp" size={18}/>Enviar digital</Button></div>
    <ActionLink tone="primary" href="/mesas">Finalizar y liberar mesa<Icon name="chevron" size={17}/></ActionLink>
  </div></div>;
}
