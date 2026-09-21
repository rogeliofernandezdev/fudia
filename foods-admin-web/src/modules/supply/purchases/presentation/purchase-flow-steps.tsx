"use client";
import {Icon} from "@/design-system";
import type {PurchaseStatus} from "../domain/types";

export function PurchaseFlowSteps({status="draft",active}:{status?:PurchaseStatus;active:"order"|"receipt"}){
  const orderDone=status==="approved"||status==="partially_received"||status==="received";
  const receiptStarted=status==="partially_received"||status==="received"||active==="receipt";
  const receiptDone=status==="received";
  return <div className="purchase-flow-steps" aria-label="Flujo de compra">
    <div className={"purchase-flow-step "+(active==="order"?"active ":"")+(orderDone?"done":"")}>
      <span className="purchase-flow-number">{orderDone?<Icon name="check" size={14}/>:1}</span>
      <span><b>Orden de compra</b><small>Define proveedor, artículos, cantidades y costo.</small></span>
    </div>
    <span className={"purchase-flow-connector "+(orderDone?"done":"")} aria-hidden="true"/>
    <div className={"purchase-flow-step "+(active==="receipt"?"active ":"")+(receiptDone?"done ":"")+(receiptStarted&&!receiptDone?"started":"")}>
      <span className="purchase-flow-number">{receiptDone?<Icon name="check" size={14}/>:2}</span>
      <span><b>Recepción</b><small>Registra lo que realmente llegó y actualiza stock.</small></span>
    </div>
  </div>;
}
